package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/DSC-PUCP/dsc-cli/internal"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6BB242")).Bold(true)
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6BB242")).Bold(true)
	purpleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#3A2E8C")).Bold(true)
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Bold(true)
)

var askCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Ask a question to the DSC assistant",
	Long:  "Ask a single question or start an interactive chat session with the DSC assistant.",
	Example: `  dsc ask "¿cuáles son las reglas del club?"
  dsc ask                                      # interactive chat`,
	Run: runAsk,
}

func init() {
	rootCmd.AddCommand(askCmd)
}

func runAsk(cmd *cobra.Command, args []string) {
	creds, err := internal.LoadCredentials()
	if err != nil || creds.GeminiKey == "" {
		fmt.Fprintln(os.Stderr, errorStyle.Render("✗")+" no Gemini API key configured")
		fmt.Fprintln(os.Stderr, dimStyle.Render("  run: dsc config set gemini-key"))
		os.Exit(1)
	}

	// Fetch docs context
	token := creds.Token
	if token == "" {
		token, _ = internal.GHToken()
	}

	var docsContext string
	if token != "" {
		fmt.Fprint(os.Stderr, dimStyle.Render("  ● Loading docs..."))
		docsContext, err = internal.FetchDocs(token)
		if err != nil {
			fmt.Fprintln(os.Stderr, "\r"+errorStyle.Render("  ✗")+" could not fetch docs: "+err.Error())
		} else {
			fmt.Fprint(os.Stderr, "\r\033[K")
		}
	}

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	// Single question mode
	if len(args) > 0 {
		question := strings.Join(args, " ")
		history := buildInitialHistory(docsContext)
		start := time.Now()
		answer, err := chatGemini(creds.GeminiKey, history, question)
		elapsed := time.Since(start)
		if err != nil {
			fmt.Fprintln(os.Stderr, errorStyle.Render("error: ")+err.Error())
			os.Exit(1)
		}
		rendered, _ := renderer.Render(answer)
		fmt.Print(rendered)
		fmt.Println(dimStyle.Render(fmt.Sprintf("  %dms", elapsed.Milliseconds())))
		return
	}

	// Interactive chat mode
	fmt.Println()
	fmt.Println("  " + headerStyle.Render("dsc") + " " + purpleStyle.Render("PUCP") + " " + dimStyle.Render("assistant"))
	if docsContext != "" {
		fmt.Println(dimStyle.Render("  docs loaded • Ctrl+C to exit"))
	} else {
		fmt.Println(dimStyle.Render("  no docs • Ctrl+C to exit"))
	}
	fmt.Println()

	history := buildInitialHistory(docsContext)
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print(promptStyle.Render("❯ "))
		if !scanner.Scan() {
			fmt.Println()
			break
		}

		question := strings.TrimSpace(scanner.Text())
		if question == "" {
			continue
		}
		if question == "exit" || question == "quit" {
			break
		}

		fmt.Println(dimStyle.Render("  thinking..."))

		start := time.Now()
		answer, err := chatGemini(creds.GeminiKey, history, question)
		elapsed := time.Since(start)
		if err != nil {
			fmt.Fprintln(os.Stderr, errorStyle.Render("  ✗ ")+err.Error())
			continue
		}

		// Clear "thinking..."
		fmt.Print("\033[1A\033[K")

		// Add to history
		history = append(history,
			map[string]any{"role": "user", "parts": []map[string]any{{"text": question}}},
			map[string]any{"role": "model", "parts": []map[string]any{{"text": answer}}},
		)

		rendered, _ := renderer.Render(answer)
		fmt.Print(rendered)
		fmt.Println(dimStyle.Render(fmt.Sprintf("  %dms", elapsed.Milliseconds())))
	}
}

func buildInitialHistory(docsContext string) []map[string]any {
	if docsContext == "" {
		return nil
	}

	systemPrompt := fmt.Sprintf(`Eres el asistente de DSC-PUCP (Developer Student Club de la Pontificia Universidad Católica del Perú). Responde basándote en la documentación oficial del club.

--- DOCUMENTACIÓN ---
%s
--- FIN DOCUMENTACIÓN ---

Responde de forma concisa y precisa en español. Si la respuesta no está en la documentación, dilo claramente.`, docsContext)

	return []map[string]any{
		{"role": "user", "parts": []map[string]any{{"text": systemPrompt}}},
		{"role": "model", "parts": []map[string]any{{"text": "Entendido. Soy el asistente de DSC-PUCP. Tengo la documentación cargada y estoy listo para responder preguntas."}}},
	}
}

func chatGemini(apiKey string, history []map[string]any, question string) (string, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=" + apiKey

	contents := make([]map[string]any, len(history))
	copy(contents, history)
	contents = append(contents, map[string]any{
		"role":  "user",
		"parts": []map[string]any{{"text": question}},
	})

	body, _ := json.Marshal(map[string]any{
		"contents": contents,
	})

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Error.Message != "" {
		return "", fmt.Errorf(result.Error.Message)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}
