package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/template"
)

var token = os.Getenv("TOKEN")

func nrand(min, max int) int {
	if min > max {
		max, min = min, max
	}

	top := big.NewInt(int64(max - min))
	n, err := rand.Int(rand.Reader, top)
	if err != nil {
		return (min + max) / 2 // Fair guess: half. What could go wrong?
	}
	return min + int(n.Int64())
}

func any(of []string) string {
	switch N := len(of); N {
	case 0:
		return ""
	case 1:
		return of[0]
	default:
		return of[nrand(0, N-1)]
	}
}

func list(of ...string) []string {
	return of
}

type response struct {
	Type string `json:"response_type"`
	Text string `json:"text"`
}

var adjectives = []string{
	"FreeBSD-based",
	"French",
	"Lovecraftian",
	"Russian",
	"Sangerian",
	"Syrian",
	"abstruse",
	"actually convex",
	"actually not at all real",
	"angered",
	"apologetic",
	"apoplectic",
	"aroused",
	"bent",
	"blue",
	"bombastic",
	"chill",
	"combustible",
	"concave",
	"confused",
	"confusing",
	"convincing",
	"dancing",
	"dead",
	"dereferenced",
	"disrespectful",
	"disturbing",
	"donkey-like",
	"drunk",
	"eldritch",
	"elongated",
	"ephemeral",
	"explosive",
	"extruded",
	"fake",
	"fiery",
	"flaccid",
	"flailing",
	"fluffy",
	"fluorescent",
	"frigid",
	"frozen",
	"funny-shaped",
	"gangrenous",
	"glorious",
	"green",
	"gummy",
	"high-velocity",
	"indescribable",
	"inert",
	"invalidated",
	"irradiated",
	"lopsided",
	"metallic",
	"naive",
	"non-manifold",
	"nuclear",
	"obtuse",
	"patriotic, but only on the internet,",
	"poofed",
	"projected",
	"psychotic",
	"purportedly convex",
	"real",
	"really damn pissed off",
	"red",
	"rugged",
	"stencilled",
	"sterling",
	"taut",
	"tight",
	"touchy-feely",
	"translucent",
	"upside-down",
	"very much alive and not at all dead",
	"viscous",
	"wooden",
}

var pluralAdjectives = append(adjectives,
	"not-racist-and-totally-have-black-friends",
)

var singularAdjectives = append(adjectives,
	"Pickman's",
	"not-racist-and-totally-has-black-friends",
)

var appendages = []string{
	"ankle",
	"arm",
	"armpit",
	"belly-button",
	"brain",
	"butt",
	"chest",
	"earhole",
	"earlobe",
	"eyesocket",
	"flappy, wobbly growth",
	"groin",
	"groinal sort of area",
	"head",
	"hidden eleventh toe",
	"kidneys",
	"knee",
	"kneecap",
	"left nipple",
	"leg",
	"liver",
	"neck",
	"nostril",
	"right nipple",
	"shoulder",
	"temple",
	"third nipple",
	"torso",
	"wrist",
	"inherited donkey",
}

var funcs = template.FuncMap{
	"bt":   func() string { return "`" },
	"rand": nrand,
	"any":  any,
	"list": list,
	"join": func(sep string, bits []string) string { return strings.Join(bits, sep) },
	"slice": func(from, to int, bits []string) ([]string, error) {
		N := len(bits)
		if from < 0 {
			from = N + from
		}
		if to < 0 {
			to = N + to
		}

		if to < 0 || to > N {
			return nil, errors.New("slice: to is out of range")
		} else if from < 0 || from > N {
			return nil, errors.New("slice: from is out of range")
		} else if to < from {
			return nil, errors.New("slice: to is before from")
		}

		return bits[from:to], nil
	},
	"adjectives": func(n int) []string {
		if n <= 1 {
			return singularAdjectives
		}
		return pluralAdjectives
	},
	"enumerate": enumerate,
}

func enumerate(andor string, things []string) string {
	andor = strings.TrimSpace(andor)
	if len(andor) > 1 {
		andor += " "
	}

	n := len(things)
	switch n {
	case 0:
		return ""
	case 1:
		return things[0]
	case 2:
		if andor == "" {
			andor = ", "
		} else {
			andor = " " + andor
		}
		return things[0] + andor + things[1]
	default:
		return strings.Join(things[:n-1], ", ") + ", " + andor + things[n-1]
	}
}

var commands = map[string]*template.Template{
	// Acceptable context is defined by the flatValues anonymous struct in handleAwk
	"help": template.Must(template.New("help").Funcs(funcs).Parse(
		`{{bt}}/awkawk [means] [victim]{{bt}} - {{bt}}[means]{{bt}} may be one of {{enumerate "or" .CommandNames}}. The {{bt}}[victim]{{bt}} is pitiable.`,
	)),
	"grackle": template.Must(template.New("grackle").Funcs(funcs).Parse(
		`impales {{ .Target }} through the {{ .Appendages | any }} with a high-velocity grackle. AWK AWK MOTHA-FUCKA!`,
	)),
	"flamingo": template.Must(template.New("flamingo").Funcs(funcs).Parse(
		`smacks {{ .Target }} upside the {{ .Appendages | any }} with a {{ adjectives 1 | any }} flamingo.`,
	)),
	"trout": template.Must(template.New("trout").Funcs(funcs).Parse(
		`slaps {{ .Target }} around a bit with a {{ adjectives 1 | any }} trout.`,
	)),
	"cat": template.Must(template.New("cat").Funcs(funcs).Parse(
		`{{ $n := rand 1 60 }}straps {{ if eq $n 1 }}a{{ else }}{{ print $n }}{{ end }} {{ adjectives $n | any }} cat{{ if ne $n 1 }}s{{ end }} to {{ .Target }}.`,
	)),
}

var cmdnames []string

func init() {
	for k := range commands {
		if k == "help" {
			continue
		}
		cmdnames = append(cmdnames, k)
	}
	sort.Strings(cmdnames)
}

func handleAwk(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(token) > 0 && token != r.FormValue("token") {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if cmd := r.FormValue("command"); cmd == "" {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "no command given")
		return
	} else if cmd != "/awkawk" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "awkawk: expected command /awkawk, got %q", cmd)
		return
	}

	flatValues := struct {
		Flat         map[string]string
		Form         map[string][]string
		Target       string
		Means        string
		CommandNames []string
		PluralAdj    []string
		SingularAdj  []string
		Adj          []string
		Appendages   []string
	}{
		Flat:         make(map[string]string, len(r.Form)),
		Form:         r.Form,
		CommandNames: cmdnames,
		PluralAdj:    pluralAdjectives,
		SingularAdj:  singularAdjectives,
		Adj:          adjectives,
		Appendages:   appendages,
	}

	for k, ary := range r.Form {
		flatValues.Flat[k] = strings.Join(ary, " ")
	}

	cmd := strings.Fields(flatValues.Flat["text"])
	if len(cmd) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "no command given")
		return
	}

	flatValues.Means = cmd[0]
	tx, ok := commands[flatValues.Means]
	if !ok {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "unknown command %q", cmd[0])
		return
	}

	flatValues.Target = strings.Join(cmd[1:], " ")

	var buf bytes.Buffer
	if err := tx.ExecuteTemplate(&buf, cmd[0], flatValues); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error: %v", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response{
		Type: "in_channel",
		Text: buf.String(),
	}); err != nil {
		log.Println("error encoding JSON:", err)
	}
}

type Always int

func (a Always) ServeHTTP(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(int(a)) }

func main() {
	http.HandleFunc("/", handleAwk)
	http.Handle("/_health", Always(http.StatusOK))

	if err := http.ListenAndServe(os.Getenv("LISTEN"), nil); err != nil {
		panic(err)
	}
}
