package board

import (
	"encoding/json"
	"strings"
)

// BuiltinTemplateIDs are stable UUIDs seeded in migration 394.
const (
	BuiltinBrainstormID = "a1000000-0000-4000-8000-000000000001"
	BuiltinExitTicketID = "a1000000-0000-4000-8000-000000000002"
	BuiltinKWLID        = "a1000000-0000-4000-8000-000000000003"
	BuiltinDiscussionID = "a1000000-0000-4000-8000-000000000004"
	BuiltinGalleryID    = "a1000000-0000-4000-8000-000000000005"
	BuiltinTimelineID   = "a1000000-0000-4000-8000-000000000006"
	BuiltinMapID        = "a1000000-0000-4000-8000-000000000007"
	BuiltinQAID         = "a1000000-0000-4000-8000-000000000008"
)

type builtinLocaleOverlay struct {
	Title       string
	Description string
	SectionTitles map[string]string // section key → title
	Posts       map[string]struct {
		Title string
		Body  string
	}
}

// Localized overlays for built-in templates (FR-9 / AC-7). Missing locales fall back to English seed.
var builtinOverlays = map[string]map[string]builtinLocaleOverlay{
	"es": {
		BuiltinBrainstormID: {
			Title:       "Muro de lluvia de ideas",
			Description: "Muro abierto para generar ideas rápidamente.",
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt": {
					Title: "Indicación",
					Body:  "¿Qué ideas tienes? Añade una tarjeta por cada pensamiento.",
				},
			},
		},
		BuiltinExitTicketID: {
			Title:       "Ticket de salida",
			Description: "Reflexión rápida al final de clase con una sola pregunta.",
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt": {
					Title: "Ticket de salida",
					Body:  "¿Qué aprendiste hoy y qué pregunta te queda?",
				},
			},
		},
		BuiltinKWLID: {
			Title:       "Tabla KWL",
			Description: "Columnas Sé / Quiero saber / Aprendí.",
			SectionTitles: map[string]string{
				"know":    "Sé",
				"want":    "Quiero saber",
				"learned": "Aprendí",
			},
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt-k": {
					Title: "¿Qué sabes ya?",
					Body:  "Añade tarjetas en Sé con conocimientos previos.",
				},
			},
		},
		BuiltinDiscussionID: {
			Title:       "Discusión",
			Description: "Muro de conversación con una pregunta guía.",
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt": {
					Title: "Pregunta de discusión",
					Body:  "Comparte tu respuesta y reacciona a dos compañeros.",
				},
			},
		},
		BuiltinGalleryID: {
			Title:       "Galería",
			Description: "Cuadrícula para mostrar imágenes y medios.",
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt": {
					Title: "Brief de galería",
					Body:  "Sube una imagen o archivo que represente tu trabajo. Añade un pie breve.",
				},
			},
		},
		BuiltinTimelineID: {
			Title:       "Línea de tiempo",
			Description: "Tablero cronológico para eventos e hitos.",
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt": {
					Title: "Construye la línea de tiempo",
					Body:  "Añade tarjetas para eventos clave y asigna una fecha a cada una.",
				},
			},
		},
		BuiltinMapID: {
			Title:       "Mapa",
			Description: "Tablero geográfico para actividades por lugar.",
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt": {
					Title: "Marca un lugar",
					Body:  "Añade una tarjeta y establece coordenadas de un lugar relevante.",
				},
			},
		},
		BuiltinQAID: {
			Title:       "Preguntas y respuestas",
			Description: "Muro de preguntas con votos.",
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt": {
					Title: "Haz una pregunta",
					Body:  "Publica una pregunta. Vota las que quieras responder primero.",
				},
			},
		},
	},
	"fr": {
		BuiltinBrainstormID: {
			Title:       "Mur de remue-méninges",
			Description: "Mur ouvert pour générer des idées rapidement.",
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt": {
					Title: "Consigne",
					Body:  "Quelles idées avez-vous ? Ajoutez une carte pour chaque idée.",
				},
			},
		},
		BuiltinExitTicketID: {
			Title:       "Ticket de sortie",
			Description: "Réflexion rapide de fin de cours avec une seule question.",
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt": {
					Title: "Ticket de sortie",
					Body:  "Qu’avez-vous appris aujourd’hui, et quelle question vous reste-t-il ?",
				},
			},
		},
		BuiltinKWLID: {
			Title:       "Tableau KWL",
			Description: "Colonnes Je sais / Je veux savoir / J’ai appris.",
			SectionTitles: map[string]string{
				"know":    "Je sais",
				"want":    "Je veux savoir",
				"learned": "J’ai appris",
			},
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt-k": {
					Title: "Que savez-vous déjà ?",
					Body:  "Ajoutez des cartes sous Je sais pour les connaissances antérieures.",
				},
			},
		},
		BuiltinDiscussionID: {
			Title:       "Discussion",
			Description: "Mur de discussion avec une question guide.",
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt": {
					Title: "Question de discussion",
					Body:  "Partagez votre réponse, puis réagissez à deux camarades.",
				},
			},
		},
		BuiltinGalleryID: {
			Title:       "Galerie",
			Description: "Grille pour présenter images et médias.",
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt": {
					Title: "Brief de galerie",
					Body:  "Téléversez une image ou un fichier qui représente votre travail. Ajoutez une courte légende.",
				},
			},
		},
		BuiltinTimelineID: {
			Title:       "Chronologie",
			Description: "Tableau chronologique pour événements et jalons.",
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt": {
					Title: "Construire la chronologie",
					Body:  "Ajoutez des cartes pour les événements clés et définissez la date de chaque carte.",
				},
			},
		},
		BuiltinMapID: {
			Title:       "Carte",
			Description: "Tableau géographique pour activités basées sur les lieux.",
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt": {
					Title: "Épingler un lieu",
					Body:  "Ajoutez une carte et définissez les coordonnées d’un lieu pertinent.",
				},
			},
		},
		BuiltinQAID: {
			Title:       "Questions-réponses",
			Description: "Mur de questions avec votes.",
			Posts: map[string]struct {
				Title string
				Body  string
			}{
				"prompt": {
					Title: "Poser une question",
					Body:  "Publiez une question. Votez pour celles auxquelles vous voulez une réponse en premier.",
				},
			},
		},
	},
}

// ApplyBuiltinLocale overlays localized title/description/seed copy onto a built-in template.
func ApplyBuiltinLocale(tmpl *Template, locale string) {
	if tmpl == nil || tmpl.Scope != TemplateScopeBuiltin {
		return
	}
	lang := primaryLang(locale)
	if lang == "" || lang == "en" {
		return
	}
	byID, ok := builtinOverlays[lang]
	if !ok {
		return
	}
	overlay, ok := byID[tmpl.ID]
	if !ok {
		return
	}
	if overlay.Title != "" {
		tmpl.Title = overlay.Title
	}
	if overlay.Description != "" {
		tmpl.Description = overlay.Description
	}
	def, err := ParseDefinition(tmpl.Definition)
	if err != nil {
		return
	}
	for i := range def.Sections {
		if title, ok := overlay.SectionTitles[def.Sections[i].Key]; ok && title != "" {
			def.Sections[i].Title = title
		}
	}
	for i := range def.SeedPosts {
		key := def.SeedPosts[i].Key
		if key == "" {
			continue
		}
		if p, ok := overlay.Posts[key]; ok {
			if p.Title != "" {
				def.SeedPosts[i].Title = p.Title
			}
			if p.Body != "" {
				body, _ := json.Marshal(map[string]string{"text": p.Body})
				def.SeedPosts[i].Body = body
			}
		}
	}
	if raw, err := MarshalDefinition(def); err == nil {
		tmpl.Definition = raw
	}
}

func primaryLang(locale string) string {
	locale = strings.TrimSpace(strings.ToLower(locale))
	if locale == "" {
		return ""
	}
	if i := strings.IndexAny(locale, "_-"); i > 0 {
		return locale[:i]
	}
	return locale
}
