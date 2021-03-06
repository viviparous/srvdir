package client

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
)

var defaultTemplate *template.Template

// separate on any colon that isn't followed by a backslash (thanks windows!)
var nameSeparator = regexp.MustCompile(":[^\\\\]")

func init() {
	defaultTemplate = template.Must(template.New("dirlist").Parse(`
<!doctype html>
<html>
  <body>
    <h3>Directory listing for {{.Name}}</h3>
    <pre>
{{ range .Contents }}<a href="{{.}}">{{.}}</a>
{{ end }}
    </pre>
  </body>
</html>
`))
}

type Directory struct {
	subdomain string
	path      string
}

type Options struct {
	serverAddr string
	logto      string
	auth       string
	authtoken  string
	configPath string
	tmpl       *template.Template
	readOnly   bool
	index      bool
	dirs       []Directory
}

func parseArgs() (*Options, error) {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] [name:]/path/to/serve [name:]/another/path ...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOPTIONS\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEXAMPLES\n")
		fmt.Fprintf(os.Stderr, `
%s /tmp                Serves your /tmp directory on a random subdomain of srvdir.net
%s example:/tmp        Serves your /tmp directory on example.srvdir.net
%s foo:/tmp bar:/usr   Serves /tmp on foo.srvdir.net and /usr on bar.srvdir.net
%s                     Serves the current directory on a random subdomain of srvdir.net
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
	}

	serverAddr := flag.String("serverAddr", "v1.srvdir.net:443", "Address of srvdird")
	logto := flag.String("log", "", "File to log to or 'stdout' for console")
	auth := flag.String("auth", "", "username:password HTTP basic auth creds protecting the the public file server")
	authtoken := flag.String("authtoken", "", "Authtoken which identifies a srvdir.net account")
	readOnly := flag.Bool("readonly", true, "don't handle DELETE or PUT requests")
	index := flag.Bool("index", true, "render index.html instead of directory listings")
	configPath := flag.String("config", "", "Path to srvdir configuration file, default $HOME/.srvdir")
	tmplPath := flag.String("template", "", "path to a file with a custom html template for the directory listing")
	v := flag.Bool("version", false, "print version and exit")

	flag.Parse()
	args := flag.Args()

	if *v {
		fmt.Println(version)
		os.Exit(0)
	}

	var dirs []Directory
	if len(args) == 0 {
		// default with no arguments is to serve the current working directory with a random subdomain
		dirs = []Directory{{subdomain: "", path: "."}}
	} else {
		dirs = make([]Directory, len(args))

		for i, arg := range args {
			var name, path string = "", arg

			loc := nameSeparator.FindStringIndex(arg)
			if loc != nil {
				name, path = arg[:loc[0]], arg[loc[1]-1:]
			}

			fi, err := os.Stat(path)
			if err != nil {
				return nil, fmt.Errorf("Failed to stat '%s': %v", path, err)
			}

			if !fi.IsDir() {
				return nil, fmt.Errorf("%s is not a directory", path)
			}

			dirs[i] = Directory{subdomain: name, path: path}
		}
	}

	// make paths absolute
	for i, d := range dirs {
		var err error
		dirs[i].path, err = filepath.Abs(d.path)
		if err != nil {
			return nil, fmt.Errorf("Failed to extract aboslute path for dir '%s': %v", d.path, err)
		}
	}

	var tmpl *template.Template
	if *tmplPath != "" {
		var err error
		tmpl, err = template.ParseFiles(*tmplPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse template file '%s': %v", *tmplPath, err)
		}
	} else {
		tmpl = defaultTemplate
	}

	return &Options{
		serverAddr: *serverAddr,
		logto:      *logto,
		auth:       *auth,
		authtoken:  *authtoken,
		configPath: *configPath,
		readOnly:   *readOnly,
		index:      *index,
		tmpl:       tmpl,
		dirs:       dirs,
	}, nil
}
