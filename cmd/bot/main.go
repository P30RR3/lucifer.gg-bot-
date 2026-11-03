package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"krypton.dev/internal/antiraid"
	"krypton.dev/internal/antiraid/detectors"
	"krypton.dev/internal/bot"
	"krypton.dev/internal/bot/commands/agenda"
	"krypton.dev/internal/bot/commands/ai"
	arCog "krypton.dev/internal/bot/commands/antiraid"
	auditCog "krypton.dev/internal/bot/commands/audit"
	backupCog "krypton.dev/internal/bot/commands/backup"
	"krypton.dev/internal/bot/commands/community"
	"krypton.dev/internal/bot/commands/communityflow"
	configCog "krypton.dev/internal/bot/commands/config"
	"krypton.dev/internal/bot/commands/economy"
	"krypton.dev/internal/bot/commands/engagement"
	"krypton.dev/internal/bot/commands/general"
	"krypton.dev/internal/bot/commands/luxpanel"
	"krypton.dev/internal/bot/commands/moderation"
	"krypton.dev/internal/bot/commands/sanctions"
	"krypton.dev/internal/bot/commands/utility"
	"krypton.dev/internal/bot/components"
	"krypton.dev/internal/cache"
	"krypton.dev/internal/cerebras"
	"krypton.dev/internal/config"
	appLogger "krypton.dev/internal/logger"
	"krypton.dev/internal/storage"
	"krypton.dev/internal/ui"
	"krypton.dev/internal/ui/live"
)

var (
	Version   = "1.0.0"
	Commit    = "dev"
	BuildDate = "2026"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Cargar configuración
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	// 2. Inicializar Logger Premium
	var logLevel slog.Level
	switch cfg.Log.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	appLogger.Init(logLevel, cfg.Log.Format)
	logger := slog.Default()

	logger.Info("🍸 Lucifer Morningstar abriendo las puertas del Lux...",
		"version", Version,
		"commit", Commit,
		"env", string(cfg.Env),
		"go", runtime.Version(),
		"cpus", runtime.NumCPU(),
	)

	// Contexto para señales de apagado
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	start := time.Now()

	// 3. Conectar dependencias externas en paralelo; Redis no depende de la BD.
	store, cacheClient, err := initializeDependencies(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer store.Close()
	defer func() {
		if err := cacheClient.Close(); err != nil {
			logger.Warn("⚠️ Error cerrando cache", "err", err)
		}
	}()

	// 5. Inicializar IA (Cerebras)
	aiClient := cerebras.NewClient(cfg.Cerebras, logger)
	logger.Info("🤖 Motor conversacional preparado", "provider", "Cerebras", "model", cfg.Cerebras.Model)
	batcher := cerebras.NewBatcher(aiClient, cfg.Cerebras.BatchSize, cfg.Cerebras.BatchFlushEvery)
	defer batcher.Close()

	// 6. UI y Tema
	theme := ui.NewTheme(cfg.UI)
	auditor := bot.NewAuditor(store, logger)

	// 7. Conectar a Discord
	session, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		return fmt.Errorf("discordgo: %w", err)
	}
	lockdown := antiraid.NewLockdownManager(session, 5*time.Minute, logger)

	if cfg.Discord.ShardCount > 0 {
		session.ShardCount = cfg.Discord.ShardCount
		session.ShardID = cfg.Discord.ShardID
		logger.Info("⚙️ Dividiendo el inframundo (Sharding manual)", "shard_count", cfg.Discord.ShardCount, "shard_id", cfg.Discord.ShardID)
	}

	session.Identify.Intents = discordgo.Intent(cfg.Discord.Intents)
	session.Identify.LargeThreshold = cfg.Discord.LargeThreshold
	session.StateEnabled = true
	session.State.TrackMembers = true
	session.State.TrackChannels = true
	session.State.TrackPresences = false

	liveManager := live.NewManager(session, cfg.UI.LiveFPS, cfg.UI.LiveMaxUpdates)
	defer liveManager.Close()

	// 8. Motor Antiraid
	engine := antiraid.New(cfg.Antiraid, session, store, theme, aiClient, logger)
	engine.SetLockdownManager(lockdown)
	engine.RegisterDetector(detectors.NewJoinFlood(cfg.Antiraid.JoinFloodLimit, cfg.Antiraid.WindowDuration))
	engine.RegisterDetector(detectors.NewMessageSpam(cfg.Antiraid.MessageSpamLimit, cfg.Antiraid.WindowDuration))
	engine.RegisterDetector(detectors.NewMentionSpam(cfg.Antiraid.MentionSpamLimit, cfg.Antiraid.WindowDuration))
	engine.RegisterDetector(detectors.NewLinkScan(cfg.Antiraid.LinkSpamLimit, cfg.Antiraid.WindowDuration))
	engine.RegisterDetector(detectors.NewAccountAge(cfg.Antiraid.MinAccountAge))
	engine.RegisterDetector(detectors.NewNuke(3, 30*time.Second))
	engine.Start(ctx)
	defer engine.Stop()

	// 9. Registro de Comandos (Cogs)
	registry := bot.NewRegistry()

	env := &bot.Env{
		Config:    cfg,
		Registry:  registry,
		Store:     store,
		Cache:     cacheClient,
		AI:        aiClient,
		Batcher:   batcher,
		Theme:     theme,
		Live:      liveManager,
		Session:   session,
		Auditor:   auditor,
		Log:       logger,
		Engine:    engine,
		Lockdown:  lockdown,
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		StartedAt: start,
	}

	cogs := []interface {
		Register(*bot.Registry) error
	}{
		general.New(),
		luxpanel.New(),
		utility.New(),
		moderation.New(),
		sanctions.New(),
		economy.New(),
		community.New(),
		communityflow.New(),
		agenda.New(),
		ai.New(),
		engagement.New(),
		configCog.New(),
		auditCog.New(),
		backupCog.New(),
		arCog.New(engine),
	}

	for idx, c := range cogs {
		if err := c.Register(registry); err != nil {
			return fmt.Errorf("registrando cog #%d: %w", idx, err)
		}
	}

	logger.Info("📜 Reglas del club registradas (Comandos)", "total", registry.Count(), "categories", len(registry.Categories()))

	// 10. Router y Componentes
	router := bot.NewRouter(session, env)
	components.RegisterAll(router, env)
	general.RegisterComponents(router)
	luxpanel.RegisterComponents(router)
	community.RegisterComponents(router)
	communityflow.RegisterComponents(router)
	engagement.RegisterComponents(router)
	auditCog.RegisterComponents(router)
	backupCog.RegisterComponents(router)
	configCog.RegisterComponents(router)
	economy.RegisterComponents(router)
	agenda.RegisterComponents(router)
	sanctions.RegisterComponents(router)
	sanctions.RegisterSecurityComponents(router)
	moderation.RegisterComponents(router)

	// 11. Eventos de Discord
	session.AddHandler(router.Handle)
	session.AddHandler(func(s *discordgo.Session, event *discordgo.MessageCreate) {
		communityflow.OnMessageCreate(s, event, store)
	})
	session.AddHandler(func(s *discordgo.Session, event *discordgo.MessageCreate) {
		engagement.OnMessageCreate(s, event, store, logger)
	})
	session.AddHandler(func(s *discordgo.Session, event *discordgo.VoiceStateUpdate) {
		engagement.OnVoiceStateUpdate(s, event, store, logger)
	})
	session.AddHandler(func(s *discordgo.Session, event *discordgo.GuildMemberAdd) {
		if event.User == nil || event.User.Bot {
			return
		}
		guildName := "el club"
		if guild, err := s.Guild(event.GuildID); err == nil && guild != nil && guild.Name != "" {
			guildName = guild.Name
		}
		settings, settingsErr := store.GetGuild(context.Background(), event.GuildID)
		if settingsErr != nil && !errors.Is(settingsErr, storage.ErrGuildNotFound) {
			logger.Warn("no se pudo cargar el canal de bienvenida", "guild_id", event.GuildID, "err", settingsErr)
		}
		welcomeMessage := ""
		if settingsErr == nil {
			welcomeMessage = settings.WelcomeMessage
		}
		welcomeImageURL := ""
		if settingsErr == nil {
			welcomeImageURL = settings.WelcomeImageURL
		}
		welcomeImageURL = configCog.ResolveWelcomeImageURL(welcomeImageURL, cfg.UI.WelcomeImageURL)
		embed := configCog.BuildWelcomeEmbed(theme, guildName, event.User, welcomeMessage, welcomeImageURL)
		welcomeFiles, imageErr := configCog.WelcomeImageFiles(welcomeImageURL)
		if imageErr != nil {
			logger.Error("no se pudo generar el fondo de bienvenida", "guild_id", event.GuildID, "err", imageErr)
			embed.Image = nil
		}
		if settingsErr == nil && settings.WelcomeChannel != "" {
			_, sendErr := s.ChannelMessageSendComplex(settings.WelcomeChannel, &discordgo.MessageSend{
				Embeds:          []*discordgo.MessageEmbed{embed},
				Files:           welcomeFiles,
				AllowedMentions: &discordgo.MessageAllowedMentions{},
			})
			if sendErr == nil {
				logger.Info("bienvenida pública enviada", "guild_id", event.GuildID, "user_id", event.User.ID, "channel_id", settings.WelcomeChannel)
				return
			}
			logger.Warn("no se pudo enviar la bienvenida pública; se intentará por mensaje directo",
				"guild_id", event.GuildID, "user_id", event.User.ID, "channel_id", settings.WelcomeChannel, "err", sendErr)
		}
		channel, err := s.UserChannelCreate(event.User.ID)
		if err == nil {
			directFiles := welcomeFiles
			if len(directFiles) > 0 {
				directFiles, imageErr = configCog.WelcomeImageFiles(welcomeImageURL)
				if imageErr != nil {
					logger.Error("no se pudo regenerar el fondo de bienvenida para el DM", "guild_id", event.GuildID, "err", imageErr)
					embed.Image = nil
					directFiles = nil
				}
			}
			_, err = s.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
				Embeds:          []*discordgo.MessageEmbed{embed},
				Files:           directFiles,
				AllowedMentions: &discordgo.MessageAllowedMentions{},
			})
		}
		if err != nil {
			logger.Warn("no se pudo enviar la bienvenida privada", "guild_id", event.GuildID, "user_id", event.User.ID, "err", err)
			return
		}
		logger.Info("bienvenida privada enviada", "guild_id", event.GuildID, "user_id", event.User.ID)
	})
	session.AddHandler(auditor.OnMessageCreate)
	session.AddHandler(auditor.OnMessageDelete)
	session.AddHandler(auditor.OnMessageDeleteBulk)
	session.AddHandler(auditor.OnMessageUpdate)
	session.AddHandler(auditor.OnVoiceStateUpdate)
	session.AddHandler(auditor.OnGuildMemberAdd)
	session.AddHandler(auditor.OnGuildMemberRemove)
	session.AddHandler(auditor.OnGuildMemberUpdate)
	session.AddHandler(auditor.OnGuildBanAdd)
	session.AddHandler(auditor.OnGuildBanRemove)
	session.AddHandler(auditor.OnChannelCreate)
	session.AddHandler(auditor.OnChannelUpdate)
	session.AddHandler(auditor.OnChannelDelete)
	session.AddHandler(auditor.OnThreadCreate)
	session.AddHandler(auditor.OnThreadDelete)
	session.AddHandler(auditor.OnGuildRoleCreate)
	session.AddHandler(auditor.OnGuildRoleUpdate)
	session.AddHandler(auditor.OnGuildRoleDelete)
	session.AddHandler(engine.OnMemberAdd)
	session.AddHandler(engine.OnMessageCreate)
	session.AddHandler(engine.OnChannelDelete)
	session.AddHandler(engine.OnRoleDelete)
	session.AddHandler(engine.OnGuildBanAdd)

	var presenceStart sync.Once
	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		presenceStart.Do(func() {
			go rotatePresence(ctx, s, logger)
		})
		logger.Info("😈 El Diablo ha llegado a Los Ángeles (Bot en línea)",
			"user", r.User.Username,
			"user_id", r.User.ID,
			"guilds", len(r.Guilds),
			"session_id", r.SessionID,
			"startup_duration", time.Since(start).String(),
		)
	})

	session.AddHandler(func(s *discordgo.Session, g *discordgo.GuildCreate) {
		if g.Unavailable {
			return
		}
		go func() {
			if _, err := store.GetOrCreateGuild(context.Background(), g.ID); err != nil {
				logger.Warn("⚠️ Un nuevo club se resiste (Error inicializando guild)", "guild", g.ID, "err", err)
			}
		}()
	})

	session.AddHandler(func(s *discordgo.Session, g *discordgo.GuildDelete) {
		if g.Unavailable {
			return
		}
		engine.InvalidateGuild(g.ID)
	})

	// 12. Abrir conexión con Discord
	if err := session.Open(); err != nil {
		return fmt.Errorf("abriendo sesión Discord: %w", err)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer closeCancel()
		select {
		case <-closeCtx.Done():
		default:
			_ = session.Close()
		}
	}()
	defer lockdown.Shutdown()

	logger.Info("🔗 Conexión al plano terrenal establecida (Gateway)")

	// 13. La sincronización REST puede tardar; no bloquea la disponibilidad del bot.
	if cfg.Discord.SyncOnStart {
		go func() {
			syncer := bot.NewSyncer(session, cfg.Discord.AppID, logger, cfg.Discord.SyncTimeout)
			results, err := syncer.SyncAll(ctx, registry, cfg.Discord.DevGuildID)
			if err != nil {
				logger.Error("❌ Sync de comandos falló", "err", err)
				return
			}
			for _, r := range results {
				scope := "global"
				if !r.Global {
					scope = "guild:" + r.GuildID
				}
				logger.Info("✅ Deseos sincronizados (Sync ok)", "scope", scope, "count", r.Count, "duration", r.Duration.String())
			}
		}()
	}

	// 14. Servidor HTTP de Health Check y OAuth2 Dashboard
	health := startHealthServer(cfg, session, store, cacheClient, engine, logger)
	logger.Info("🍸 El Lux está listo para recibir visitas", "startup_duration", time.Since(start).String())

	// Esperar señal de apagado
	<-ctx.Done()
	logger.Info("🛑 Cerrando el Lux, hora de volver al infierno...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer shutdownCancel()

	if err := health.Shutdown(shutdownCtx); err != nil {
		logger.Warn("⚠️ Health server shutdown", "err", err)
	}

	engine.Stop()
	lockdown.Shutdown()
	if err := session.Close(); err != nil {
		logger.Warn("⚠️ Session close", "err", err)
	}
	batcher.Close()
	liveManager.Close()

	logger.Info("🦇 Lucifer ha vuelto al infierno (Apagado limpio)", "uptime", time.Since(start).String())
	return nil
}

func rotatePresence(ctx context.Context, session *discordgo.Session, logger *slog.Logger) {
	type PresenceEntry struct {
		Name string
		Type discordgo.ActivityType
	}

	activities := []PresenceEntry{
		{Name: "🔱 Custodiando los fueros de alta soberanía", Type: discordgo.ActivityTypeWatching},
		{Name: "⚖️ Arbitrando el protocolo del senado imperial", Type: discordgo.ActivityTypeListening},
		{Name: "🍷 Degustando una gran reserva en cancillería", Type: discordgo.ActivityTypeCustom},
		{Name: "👑 Otorgando audiencias diplomáticas • /ask", Type: discordgo.ActivityTypeWatching},
		{Name: "⚜️ Inspeccionando las bóvedas soberanas • /saldo", Type: discordgo.ActivityTypeWatching},
		{Name: "📜 Sellando decretos de honor y distinción", Type: discordgo.ActivityTypeWatching},
		{Name: "💎 Presidiendo el salón de alta distinción • /ruleta", Type: discordgo.ActivityTypeCompeting},
		{Name: "🥂 Acreditando estipendios reales • /fichas", Type: discordgo.ActivityTypeListening},
		{Name: "⚔️ Salvaguardando el perímetro protocolar", Type: discordgo.ActivityTypeWatching},
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for index := 0; ; index = (index + 1) % len(activities) {
		current := activities[index]
		err := session.UpdateStatusComplex(discordgo.UpdateStatusData{
			Status: string(discordgo.StatusOnline),
			Activities: []*discordgo.Activity{{
				Name: current.Name,
				Type: current.Type,
			}},
		})
		if err != nil {
			logger.Warn("no se pudo actualizar la presencia soberana del bot", "err", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func initializeDependencies(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*storage.Store, *cache.Client, error) {
	type result struct {
		store *storage.Store
		cache *cache.Client
		err   error
	}

	startupCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan result, 2)

	go func() {
		start := time.Now()
		store, err := storage.New(startupCtx, cfg.Postgres, logger)
		if err == nil && cfg.Postgres.RunMigrations {
			migrationStart := time.Now()
			err = store.RunMigrations(startupCtx)
			if err == nil {
				logger.Info("🔥 Migraciones aplicadas", "duration", time.Since(migrationStart).String())
			}
		}
		if err != nil {
			if store != nil {
				store.Close()
			}
			results <- result{err: fmt.Errorf("storage: %w", err)}
			return
		}
		logger.Info("postgres listo", "duration", time.Since(start).String())
		results <- result{store: store}
	}()

	go func() {
		start := time.Now()
		client, err := cache.New(startupCtx, cfg.Redis, logger)
		if err != nil {
			results <- result{err: fmt.Errorf("cache: %w", err)}
			return
		}
		logger.Info("redis listo", "duration", time.Since(start).String())
		results <- result{cache: client}
	}()

	var store *storage.Store
	var cacheClient *cache.Client
	var startupErr error
	for range 2 {
		result := <-results
		if result.err != nil {
			if startupErr == nil {
				startupErr = result.err
				cancel()
			}
			continue
		}
		if result.store != nil {
			store = result.store
		}
		if result.cache != nil {
			cacheClient = result.cache
		}
	}
	if startupErr != nil {
		if store != nil {
			store.Close()
		}
		if cacheClient != nil {
			if err := cacheClient.Close(); err != nil {
				logger.Warn("error cerrando Redis tras fallo de inicio", "err", err)
			}
		}
		return nil, nil, startupErr
	}
	return store, cacheClient, nil
}

// --- HEALTH & OAUTH2 SERVER ---

type healthServer struct {
	server *http.Server
}

func (h *healthServer) Shutdown(ctx context.Context) error {
	if h == nil || h.server == nil {
		return nil
	}
	return h.server.Shutdown(ctx)
}

func startHealthServer(
	cfg *config.Config,
	session *discordgo.Session,
	store *storage.Store,
	cacheClient *cache.Client,
	engine *antiraid.Engine,
	logger *slog.Logger,
) *healthServer {
	mux := http.NewServeMux()

	// 1. Health check básico
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":    "ok",
			"version":   Version,
			"commit":    Commit,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// 2. Health check de dependencias
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		checks := map[string]string{}
		status := "ok"

		if err := store.Health(ctx); err != nil {
			checks["postgres"] = "fail: " + err.Error()
			status = "degraded"
		} else {
			checks["postgres"] = "ok"
		}

		if err := cacheClient.Health(ctx); err != nil {
			checks["redis"] = "fail: " + err.Error()
			status = "degraded"
		} else {
			checks["redis"] = "ok"
		}

		if session != nil && session.State != nil && session.State.User != nil {
			checks["discord"] = "ok"
		} else {
			checks["discord"] = "not ready"
			status = "degraded"
		}

		if status == "degraded" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":    status,
			"checks":    checks,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// 🔴 3. DISCORD OAUTH2: LOGIN (Redirige al portal de Discord)
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		clientID := os.Getenv("DISCORD_CLIENT_ID")
		if clientID == "" && cfg != nil {
			clientID = cfg.Discord.AppID
		}
		redirectURI := "https://lucifer-gg-bot.onrender.com/api/auth/callback"
		discordAuthURL := fmt.Sprintf(
			"https://discord.com/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=identify",
			clientID,
			url.QueryEscape(redirectURI),
		)
		http.Redirect(w, r, discordAuthURL, http.StatusTemporaryRedirect)
	})

	// 🔴 4. DISCORD OAUTH2: CALLBACK (Verifica permisos y devuelve al dashboard)
	mux.HandleFunc("/api/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		frontendURL := os.Getenv("FRONTEND_URL")
		if frontendURL == "" {
			frontendURL = "https://lucifer-gg-bot-1.onrender.com"
		}

		if code == "" {
			http.Redirect(w, r, frontendURL+"?error=access_denied", http.StatusTemporaryRedirect)
			return
		}

		clientID := os.Getenv("DISCORD_CLIENT_ID")
		if clientID == "" && cfg != nil {
			clientID = cfg.Discord.AppID
		}

		// A. Canjear código por Access Token
		data := url.Values{}
		data.Set("client_id", clientID)
		data.Set("client_secret", os.Getenv("DISCORD_CLIENT_SECRET"))
		data.Set("grant_type", "authorization_code")
		data.Set("code", code)
		data.Set("redirect_uri", "https://lucifer-gg-bot.onrender.com/api/auth/callback")

		tokenReq, _ := http.NewRequest("POST", "https://discord.com/api/oauth2/token", strings.NewReader(data.Encode()))
		tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		httpClient := &http.Client{Timeout: 10 * time.Second}
		tokenResp, err := httpClient.Do(tokenReq)
		if err != nil || tokenResp.StatusCode != http.StatusOK {
			logger.Error("error canjeando token oauth2", "err", err)
			http.Redirect(w, r, frontendURL+"?error=token_failed", http.StatusTemporaryRedirect)
			return
		}
		defer tokenResp.Body.Close()

		var tokenData struct {
			AccessToken string `json:"access_token"`
		}
		if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil || tokenData.AccessToken == "" {
			http.Redirect(w, r, frontendURL+"?error=invalid_token", http.StatusTemporaryRedirect)
			return
		}

		// B. Obtener usuario @me
		userReq, _ := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
		userReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)
		userResp, err := httpClient.Do(userReq)
		if err != nil || userResp.StatusCode != http.StatusOK {
			logger.Error("error obteniendo usuario discord @me", "err", err)
			http.Redirect(w, r, frontendURL+"?error=user_failed", http.StatusTemporaryRedirect)
			return
		}
		defer userResp.Body.Close()

		var dUser struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Avatar   string `json:"avatar"`
		}
		if err := json.NewDecoder(userResp.Body).Decode(&dUser); err != nil {
			http.Redirect(w, r, frontendURL+"?error=decode_user_failed", http.StatusTemporaryRedirect)
			return
		}

		// C. VERIFICACIÓN DE AUTORIDAD (Owner o Staff con permisos de Admin / Ban)
		guildID := "1289371089201987654"
		isStaff := false

		if session != nil {
			// Si es el bot owner en config
			if cfg != nil && cfg.Discord.BotOwnerID != "" && dUser.ID == cfg.Discord.BotOwnerID {
				isStaff = true
			} else if guild, gErr := session.Guild(guildID); gErr == nil && guild != nil && guild.OwnerID == dUser.ID {
				// Si es el dueño del servidor
				isStaff = true
			} else if member, mErr := session.GuildMember(guildID, dUser.ID); mErr == nil && member != nil {
				// Revisar roles con permisos de Administrador o Banear
				for _, roleID := range member.Roles {
					if role, rErr := session.State.Role(guildID, roleID); rErr == nil && role != nil {
						if role.Permissions&discordgo.PermissionAdministrator != 0 || role.Permissions&discordgo.PermissionBanMembers != 0 {
							isStaff = true
							break
						}
					}
				}
			}
		}

		// D. Empaquetar y redirigir al Dashboard
		payload := map[string]any{
			"id":       dUser.ID,
			"username": dUser.Username,
			"avatar":   fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", dUser.ID, dUser.Avatar),
			"isStaff":  isStaff,
		}
		payloadJSON, _ := json.Marshal(payload)
		encoded := base64.URLEncoding.EncodeToString(payloadJSON)

		http.Redirect(w, r, fmt.Sprintf("%s/?auth_data=%s", frontendURL, encoded), http.StatusTemporaryRedirect)
	})

	server := &http.Server{
		Addr:         cfg.Health.Addr,
		Handler:      mux,
		ReadTimeout:  cfg.Health.ReadTimeout,
		WriteTimeout: cfg.Health.WriteTimeout,
	}

	go func() {
		logger.Info("🏥 Servicio de bienestar y salud del Lux activo", "addr", cfg.Health.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("❌ Health server error", "err", err)
		}
	}()

	return &healthServer{server: server}
}