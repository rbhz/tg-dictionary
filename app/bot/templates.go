package bot

const DictionaryItemTemplate = `<b>{{ .Item.Word }}</b>
{{- if .Item.Translations }}
<b>Translations</b>:
{{- range $t := .Item.Translations }}
<code>{{ $t.Text }}</code> ({{ $t.Language }})
{{- end }}

{{- end }}
<b>Meanings:</b>
{{- range $m := .Item.Meanings }}
<code>{{ $m.Definition }}</code> ({{ $m.PartOfSpeech }})
{{- range $e := $m.Examples }}
{{ $e }}
{{- end }}
___
{{- end }}
<u>Phonetics</u>: {{ .Item.Phonetics.Text }}
`
