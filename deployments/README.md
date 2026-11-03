<div align="center">

# 🛡️ Krypton

**Bot de seguridad, moderación avanzada, antiraid y automatización para Discord con IA integrada.**

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![Discord](https://img.shields.io/badge/Discord-discordgo-5865F2?style=for-the-badge&logo=discord&logoColor=white)](https://github.com/bwmarrin/discordgo)
[![Cerebras](https://img.shields.io/badge/AI-Cerebras-FF6B35?style=for-the-badge)](https://cerebras.ai)
[![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)](LICENSE)

_Arquitectura limpia · Event-driven · Live messages · IA clasificadora_

</div>

---

## ✨ Características

- **🛡️ Antiraid en tiempo real** — detectores especializados (`joinflood`, `messagespam`, `mentionspam`, `linkscan`, `accountage`, `nuke`) coordinados por un motor de scoring con ventanas deslizantes.
- **🤖 Moderación asistida por IA** — clasificación estructurada de mensajes vía Cerebras (`category`, `severity`, `confidence`, `suggested_action`) con batching y rate limiting.
- **⚡ Live messages** — embeds premium que se editan incrementalmente con barras de progreso dinámicas, botones y select menus nativos.
- **🔒 Lockdown inteligente** — bloqueo automático de canales con cooldown y auto-restore programado.
- **📊 Dashboard y observabilidad** — logs estructurados (`log/slog`), métricas de rate limit y estadísticas en vivo por guild.
- **🧩 Comandos modulares (cogs-style)** — economía del Lux, propuestas comunitarias, agenda con aforo, moderación, IA y utilidades se registran por dominio.
- **💡 Tablero comunitario** — panel interactivo para consultar, votar y resolver propuestas persistentes.
- **🗓️ Agenda del Lux** — panel privado para crear eventos, ver fechas, reservar plazas y cancelar.
- **🌍 Multi-guild y sharding-ready** — configuración por guild persistida en PostgreSQL, caché en Redis, sharding nativo de `discordgo`.

---

## 🍸 Comandos de comunidad y eventos

- **Guía:** `/help` muestra una sección a la vez mediante un menú desplegable y refleja los comandos registrados. `/ayuda` abre una solicitud de soporte.
- **Canales:** el canal público de ayuda se crea automáticamente al usar `/ayuda` si aún no existe; el confesionario también se crea al usar `/deseo-profundo`. `/soporte llamar` crea su canal privado al primer uso si no hay uno configurado.
- **Colores de los embeds:** borgoña `#8B0000` para paneles Lux, ayuda, juicios, investigación y tienda; naranja `#E67E22` para redención; morado oscuro `#24152E` para mensajes neutrales/anónimos; verde `#2ECC71` para éxito y rojo `#E74C3C` para errores.
- Todas las respuestas, embeds, mensajes directos, avisos automáticos, botones, opciones de comandos y descripciones pasan por una identidad visual compartida: borgoña, detalles del Lux y un cierre de inspiración italiana, carismático y elegante. Se conserva la fuente estándar de Discord para facilitar la lectura y la accesibilidad.
- `/lux` abre un panel con pestañas de perfil, inventario, club social y ajustes. Desde ahí se consulta el saldo, se reclama la recompensa diaria, se activa la búsqueda de grupo y se guardan preferencias de idioma, avisos e inventario. Cada reclamo diario también suma 25 XP; el nivel avanza cada 100 XP. La preferencia para ocultar el inventario queda lista para perfiles públicos.
- En `/lux` → **Ajustes**, el botón **Visibilidad** hace públicas las respuestas normales del bot de forma predeterminada o las limita a quien las solicitó. Los errores siguen siendo privados. Al cambiar de modo, el panel se vuelve a publicar con la visibilidad elegida. La visibilidad pública también puede mostrar datos personales de perfil e historial, incluidos los de `/terapia`.
- `/ayuda titulo descripcion` crea un hilo con botones para marcarlo solucionado o pedir seguimiento. Al resolverlo, el staff con más respuestas recibe +1 en el ranking y una moneda Lux.
- `/demanda usuario motivo` abre un juicio privado de roleplay de 10 minutos. La IA publica un veredicto; un veredicto culpable aplica un timeout limitado a 10 minutos. Es entretenimiento y no sustituye una denuncia real.
- `/deseo-profundo` crea automáticamente el confesionario si hace falta, presenta un modal y publica el deseo de forma anónima. La autoría se conserva en PostgreSQL por 30 días. `/chantaje id precio deseo` evalúa la oferta ficticia y nunca revela al autor ni genera pistas personales.
- `/terapia` abre un centro de actividad y confidencialidad que sigue la visibilidad configurada en `/lux`. Muestra mensajes agregados, hora más activa, canal favorito, tiempo en voz y palabras frecuentes; no guarda el texto de los mensajes. El seguimiento está desactivado inicialmente y las estadísticas son privadas para los rankings; el historial completo que muestra `/terapia` será visible en el canal si las respuestas están configuradas como públicas. Desactivar el seguimiento detiene nuevas mediciones, pero conserva las anteriores.
- `/celestial` reúne misiones diarias/semanales, clasificación de texto/voz/ayuda y una tienda de roles. Las misiones incluyen 50 mensajes diarios, visitar tres canales de voz y 250 mensajes semanales; los puntos se reclaman con botones. La clasificación solo incluye miembros que hicieron públicas sus estadísticas. Administradores configuran roles canjeables con `/celestial-configurar rol`; el bot debe tener **Gestionar roles** y estar por encima del rol ofrecido. Para beneficios de adjuntos, emojis o canales, configura los permisos del rol desde Discord.
- `/panel-staff usuario` muestra el expediente privado, evidencias e interrogatorios voluntarios; cada acción comprueba el permiso real de Discord.
- `/puertas-infernales` abre la consola privada de seguridad: lista baneos y silencios activos, busca el historial por ID, revisa apelaciones y activa un bloqueo de emergencia de cinco minutos en canales de texto. Al desbloquear, restaura los permisos que tenían esos canales antes del cierre.
- `/infierno usuario minutos motivo` aplica un timeout de hasta 24 horas y abre un hilo privado con una tarea de redención. La tarea también se envía por DM para que el usuario pueda completarla aunque el timeout bloquee sus interacciones en el servidor. `/inframundo menu` centraliza los expedientes, las notas y la absolución anticipada.
- `/tienda` permite comprar ocho artículos iniciales y otros añadidos por el staff; `/tienda-admin articulo`, `/tienda-admin precio` y `/tienda-admin rango` personalizan el catálogo y los rangos por servidor.
- `/ideas enviar` publica propuestas; `/ideas panel` permite consultar, votar y resolverlas con controles interactivos. La identidad se oculta en la publicación cuando se elige anonimato.
- `/agenda panel` centraliza la creación, consulta, reserva y cancelación de eventos. Crear requiere el permiso de Discord **Gestionar eventos**; cancelar permite al organizador o al equipo autorizado.
- `/backup-panel` permite a administradores guardar y eliminar un respaldo estructural del servidor. Solo el dueño puede confirmar su carga; esta reemplaza roles y canales recreables tras una confirmación explícita. Incluye categorías, permisos, ajustes compatibles y referencias de canales especiales/de bienvenida/auditoría/ayuda, pero no mensajes, miembros, asignaciones de roles, hilos, integraciones ni iconos de rol. La creación no sobrescribe un respaldo existente; elimínalo desde el panel para guardar otro. Requiere PostgreSQL y que el bot pueda administrar roles y canales; si el respaldo incluye canales especiales del servidor, también necesita Administrar servidor.
- Los saldos, propuestas, votos, eventos e inscripciones se guardan en PostgreSQL. Las migraciones nuevas se aplican al iniciar si `POSTGRES_RUN_MIGRATIONS=true`.
- El inicio conecta PostgreSQL y Redis en paralelo y aplica las migraciones mientras Redis se prepara. La conexión de Discord y el servidor de salud no esperan la sincronización REST de comandos; la consola registra su duración por separado.

---

## 📜 Auditoría, soporte y guardia

- `/logs configurar canal contenido:false` activa el diario por servidor; `/logs estado` y `/logs prueba` permiten revisarlo. Los registros cubren mensajes editados/eliminados, actividad de voz, miembros, bans, canales, hilos y roles. El bot necesita permiso para ver y escribir en el canal, y para insertar embeds.
- `/logs eliminados horas:6` muestra de forma efímera hasta cinco mensajes recientes. El contenido se conserva **solo si** se activa la opción `contenido`; los registros se eliminan después de siete días. Un caché en memoria, limitado a 5.000 mensajes, permite capturar contenido cuando está habilitado. Los mensajes directos no se registran.
- `/soporte llamar motivo` notifica roles con permisos de moderación en el canal privado de guardia (creado automáticamente si hace falta), con una espera de cinco minutos por usuario.
- `/anuncio prompt tono` genera un borrador privado (chill, elegante o urgente) para que el staff lo revise y publique manualmente. El bot neutraliza menciones masivas y no publica anuncios por su cuenta.
- `/advertir usuario razon` registra una advertencia, intenta notificar al usuario por DM y la incluye en el diario y el ranking. El aviso incluye un formulario privado de apelación; las apelaciones se guardan y aparecen en `/puertas-infernales`.
- `/staff periodo` muestra un ranking de hasta diez personas con advertencias, bans, expulsiones, silencios y limpiezas ejecutados mediante los comandos de Krypton. No es posible reconstruir acciones anteriores a la instalación del registro.
- Los nuevos módulos de comunidad y moderación requieren que las migraciones PostgreSQL estén activadas (`POSTGRES_RUN_MIGRATIONS=true`). Los hilos privados requieren permisos de gestión de hilos; los juicios requieren permiso de timeout y disponibilidad de la IA.
- Los paneles `/terapia` y `/celestial` usan la migración `0008_member_engagement`; la preferencia de visibilidad de `/lux` se agrega con `0009_lux_response_visibility`. Las estadísticas no se reconstruyen antes de instalar esta versión y se almacenan por servidor; se requiere el intent de voz para medir visitas y tiempo en canales de voz.
- `/ask` conversa con Lucifer mediante Cerebras. Las respuestas de IA son texto; las cargas y errores se muestran en embeds. `/anuncio` genera un borrador de texto para revisar; el bot no lo publica automáticamente.
- `/botstats` está reservado al dueño del bot. Configura `BOT_OWNER_ID` con el ID de usuario del propietario en el entorno del proceso.
- Los logs de consola usan niveles y colores ANSI cuando stdout es una terminal compatible; se eliminan emojis decorativos para que CMD no muestre símbolos corruptos y se conservan los acentos. El modo JSON se conserva con `LOG_FORMAT=json`. La bienvenida de nuevos miembros se envía por DM de forma predeterminada; `/configuracion` permite elegir un canal público, configurar auditoría y personalizar el mensaje con `{usuario}` y `{servidor}`. Incluye vista previa privada, envío de prueba y configuración de una imagen por servidor mediante una URL HTTPS directa. Sin imagen personalizada ni `WELCOME_IMAGE_URL`, el bot adjunta un fondo infernal generado por el propio bot y muestra el avatar redondo del usuario. La imagen de servidor prevalece sobre la URL global. La columna de imagen se agrega con la migración `0010_guild_welcome_image`; las migraciones PostgreSQL deben estar activadas.
- La IA utiliza `CEREBRAS_MODEL=gpt-oss-120b` por defecto, modelo incluido en el catálogo actual de Cerebras Shared Inference. Si el archivo `.env` fija otro modelo, asegúrate de que sigue disponible para tu cuenta.
- Las tablas de auditoría se crean con la migración `0004_audit_staff`. Los eventos de Gateway dependen de sus intents: habilita **Server Members Intent** y **Message Content Intent** en el portal de Discord para los eventos y el texto correspondientes. La configuración de intents debe incluir también estados de voz. Los mensajes eliminados en grupo no incluyen contenido en el evento de Discord.

---

## 🏗️ Estructura del proyecto
"# lucifer.gg-bot-" 
