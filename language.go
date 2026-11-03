package main

import "fmt"

// inLanguage leaves the original request alone unless a language was chosen.
// Put the instruction before any transcript payload: that payload is data,
// never a place to add instructions for a summarizer.
func inLanguage(prompt, language string) string {
	if language == "" {
		return prompt
	}
	return fmt.Sprintf("Respond in %q. Preserve code, commands, paths, proper names and ticket keys.\n\n%s", language, prompt)
}
