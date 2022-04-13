package bot

const DictionaryItemTemplate = `<b>{{ .Item.Word }}</b>
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
