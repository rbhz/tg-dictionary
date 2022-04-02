package bot

const DictionaryItemTemplate = `<b>{{ .Item.Word }}</b>
<u>Phonetics</u>: {{ .Item.Phonetics.Text }}

<b>Meanings:</b>
{{- range $m := .Item.Meanings }}
<code>{{ $m.Definition }} ({{ $m.PartOfSpeech }})</code>
{{- range $e := $m.Examples }}
{{ $e }}
{{- end }}
___
{{- end }}
`
