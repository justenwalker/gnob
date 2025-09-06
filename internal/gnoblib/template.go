package gnoblib

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/template"
	"unicode"
)

type _template struct {
}

// WriteFile executes the template and writes it to the target file, using the given data
func (t _template) WriteFile(target string, mode os.FileMode, tp *template.Template, data any) error {
	out, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("unable to create target file %q: %w", target, err)
	}
	if err = tp.Execute(out, data); err != nil {
		_ = out.Close()
		_ = os.Remove(target)
		return fmt.Errorf("unable to execute template: %w", err)
	}
	if err = out.Close(); err != nil {
		_ = os.Remove(target)
		return fmt.Errorf("unable to close target file %q: %w", target, err)
	}
	return nil
}

// ParseText parses a Go template from the text.
func (t _template) ParseText(text string) (*template.Template, error) {
	return t.ParseTextFuncs(text, nil)
}

// ParseTextFuncs parses a Go template from the text, using an extra set of template functions.
func (t _template) ParseTextFuncs(text string, extraFuncs template.FuncMap) (*template.Template, error) {
	tt := t.newTemplate(extraFuncs)
	tt, err := tt.Parse(text)
	if err != nil {
		return nil, err
	}
	return tt, nil
}

// ParseFile parses a Go template from the given file.
func (t _template) ParseFile(templateFile string) (*template.Template, error) {
	return t.ParseFileFuncs(templateFile, nil)
}

// ParseFileFuncs parses a Go template from the given file, using an extra set of template functions.
func (t _template) ParseFileFuncs(templateFile string, extraFuncs template.FuncMap) (*template.Template, error) {
	tf, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read template file %q: %w", templateFile, err)
	}
	tt, err := t.ParseTextFuncs(string(tf), extraFuncs)
	if err != nil {
		return nil, fmt.Errorf("unable to parse template file %q: %w", templateFile, err)
	}
	return tt, nil
}

func (t _template) newTemplate(extraFuncs template.FuncMap) *template.Template {
	return template.New("").Funcs(t.funcMap()).Funcs(extraFuncs)
}

func (t _template) funcMap() template.FuncMap {
	funcMap := make(template.FuncMap)
	// includeFile loads a file by path into the template
	funcMap["includeFile"] = func(path string) (string, error) {
		fs, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("unable to read file %q: %w", path, err)
		}
		return string(fs), nil
	}
	// includeFileRegion includes a portion of a file that is surrounded by a region comment.
	funcMap["includeFileRegion"] = func(path string, region string) (string, error) {
		fd, err := os.Open(path)
		if err != nil {
			return "", fmt.Errorf("unable to open file %q: %w", path, err)
		}
		defer fd.Close()
		var sb strings.Builder
		bs := bufio.NewScanner(fd)
		var inRegion bool
		for bs.Scan() {
			if strings.Contains(bs.Text(), region) {
				inRegion = !inRegion
				if !inRegion {
					break
				}
				continue
			}
			if inRegion {
				sb.WriteString(bs.Text())
				sb.WriteRune('\n')
			}
		}
		if err = bs.Err(); err != nil {
			return "", fmt.Errorf("unable to extract region: %w", err)
		}
		return sb.String(), nil
	}
	indentInternal := func(s string, n int, skip bool) (string, error) {
		var sb strings.Builder
		bs := bufio.NewScanner(strings.NewReader(s))
		var i int
		for bs.Scan() {
			i++
			if i != 1 || !skip {
				sb.WriteString(strings.Repeat(" ", n))
			}
			sb.WriteString(bs.Text())
			sb.WriteRune('\n')
		}
		if err := bs.Err(); err != nil {
			return "", fmt.Errorf("unable to indent: %w", err)
		}
		return sb.String(), nil
	}
	// indent all lines
	funcMap["indent"] = func(n int, s string) (string, error) {
		return indentInternal(s, n, false)
	}
	// indent all lines except the first
	funcMap["nindent"] = func(n int, s string) (string, error) {
		return indentInternal(s, n, true)
	}
	// unindent strips the leading 'n' characters of whitespace from all lines
	funcMap["unindent"] = func(n int, s string) (string, error) {
		var sb strings.Builder
		bs := bufio.NewScanner(strings.NewReader(s))
		for bs.Scan() {
			text := bs.Text()
			for i, r := range text {
				if !unicode.IsSpace(r) || i >= n {
					sb.WriteString(text[i:])
					break
				}
			}
			sb.WriteRune('\n')
		}
		if err := bs.Err(); err != nil {
			return "", fmt.Errorf("unable to unindent: %w", err)
		}
		return sb.String(), nil
	}
	// prepend all lines with the given prefix string
	funcMap["prefix"] = func(prefix string, s string) (string, error) {
		var sb strings.Builder
		bs := bufio.NewScanner(strings.NewReader(s))
		for bs.Scan() {
			sb.WriteString(prefix)
			sb.WriteString(bs.Text())
			sb.WriteRune('\n')
		}
		if err := bs.Err(); err != nil {
			return "", fmt.Errorf("unable to prefix: %w", err)
		}
		return sb.String(), nil
	}
	// includeTemplate parses and executes the template, passing the given data.
	funcMap["includeTemplate"] = func(path string, data any) (string, error) {
		tt, err := t.ParseFile(path)
		if err != nil {
			return "", err
		}
		var sb strings.Builder
		if err = tt.Execute(&sb, data); err != nil {
			return "", err
		}
		return sb.String(), nil
	}
	return funcMap
}
