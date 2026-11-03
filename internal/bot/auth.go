package bot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type DiscordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Avatar        string `json:"avatar"`
	Discriminator string `json:"discriminator"`
}

type AuthPayload struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	IsStaff  bool   `json:"isStaff"`
}

// Iniciar sesión: Redirige a Discord
func HandleDiscordLogin(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("DISCORD_CLIENT_ID")
	redirectURI := "https://lucifer-gg-bot.onrender.com/api/auth/callback"
	
	discordAuthURL := fmt.Sprintf(
		"https://discord.com/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=identify",
		clientID,
		url.QueryEscape(redirectURI),
	)

	http.Redirect(w, r, discordAuthURL, http.StatusTemporaryRedirect)
}

// Callback: Discord devuelve el código de acceso
func HandleDiscordCallback(w http.ResponseWriter, r *http.Request, s *discordgo.Session) {
	code := r.URL.Query().Get("code")
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "https://lucifer-gg-bot-1.onrender.com"
	}

	if code == "" {
		http.Redirect(w, r, frontendURL+"?error=access_denied", http.StatusTemporaryRedirect)
		return
	}

	// 1. Canjear el código por un token de acceso
	data := url.Values{}
	data.Set("client_id", os.Getenv("DISCORD_CLIENT_ID"))
	data.Set("client_secret", os.Getenv("DISCORD_CLIENT_SECRET"))
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", "https://lucifer-gg-bot.onrender.com/api/auth/callback")

	req, _ := http.NewRequest("POST", "https://discord.com/api/oauth2/token", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Redirect(w, r, frontendURL+"?error=token_exchange_failed", http.StatusTemporaryRedirect)
		return
	}
	defer resp.Body.Close()

	var tokenRes map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokenRes)
	accessToken, ok := tokenRes["access_token"].(string)
	if !ok {
		http.Redirect(w, r, frontendURL+"?error=invalid_token", http.StatusTemporaryRedirect)
		return
	}

	// 2. Obtener los datos del usuario en Discord
	reqUser, _ := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
	reqUser.Header.Set("Authorization", "Bearer "+accessToken)

	respUser, err := client.Do(reqUser)
	if err != nil || respUser.StatusCode != http.StatusOK {
		http.Redirect(w, r, frontendURL+"?error=failed_user_fetch", http.StatusTemporaryRedirect)
		return
	}
	defer respUser.Body.Close()

	bodyUser, _ := io.ReadAll(respUser.Body)
	var discordUser DiscordUser
	json.Unmarshal(bodyUser, &discordUser)

	// 3. Revisar si es Staff en tu servidor de Discord
	guildID := "1289371089201987654" // ID de tu servidor Lucifer Protocol
	isStaff := false

	if s != nil {
		member, err := s.GuildMember(guildID, discordUser.ID)
		if err == nil && member != nil {
			// El bot revisa si el usuario tiene permisos de administrador o si es dueño
			guild, gErr := s.Guild(guildID)
			if gErr == nil && guild.OwnerID == discordUser.ID {
				isStaff = true
			} else {
				// Revisar permisos de roles
				for _, roleID := range member.Roles {
					role, rErr := s.State.Role(guildID, roleID)
					if rErr == nil && (role.Permissions&discordgo.PermissionAdministrator != 0 || role.Permissions&discordgo.PermissionBanMembers != 0) {
						isStaff = true
						break
					}
				}
			}
		}
	}

	// 4. Empaquetar datos y devolver al usuario a la web
	payload := AuthPayload{
		ID:       discordUser.ID,
		Username: discordUser.Username,
		Avatar:   fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", discordUser.ID, discordUser.Avatar),
		IsStaff:  isStaff,
	}

	payloadJSON, _ := json.Marshal(payload)
	encodedPayload := base64.URLEncoding.EncodeToString(payloadJSON)

	// Redirigir de regreso al frontend con su sesión lista
	http.Redirect(w, r, fmt.Sprintf("%s/?auth_data=%s", frontendURL, encodedPayload), http.StatusTemporaryRedirect)
}