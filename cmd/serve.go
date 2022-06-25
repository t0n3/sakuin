/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/t0n3/sakuin/web"
)

type fileItem struct {
	Name  string
	Size  string
	Date  string
	IsDir bool
	Path  string
}

type templateVariables struct {
	Path  []breadcrumb
	Files []fileItem
}

type breadcrumb struct {
	Name string
	Path string
}

var dataDir string

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		dataDir = "tests"
		log.Println("Starting Sakuin HTTP Server")
		mux := http.NewServeMux()
		mux.Handle("/assets/", web.AssetsHandler("/assets/", "dist"))
		mux.HandleFunc("/", serve)
		http.ListenAndServe(":3000", mux)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func serve(w http.ResponseWriter, r *http.Request) {
	// Filepath, from the root data dir
	fp := filepath.Join(dataDir, filepath.Clean(r.URL.Path))
	// Cleaned filepath, without the root data dir, used for template rendering purpose
	cfp := strings.Replace(fp, dataDir, "", 1)

	// Return a 404 if the template doesn't exist
	info, err := os.Stat(fp)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			log.Printf("404 - %s\n", cfp)
			return
		}
	}

	// Return a 404 if the request is for a directory
	if info.IsDir() {
		files, err := ioutil.ReadDir(fp)
		if err != nil {
			log.Fatal(err)
		}

		// Init template variables
		templateVars := templateVariables{}

		// Construct the breadcrumb
		path := strings.Split(cfp, "/")
		for len(path) > 1 {
			b := breadcrumb{
				Name: path[len(path)-1],
				Path: strings.Join(path, "/"),
			}
			path = path[:len(path)-1]
			templateVars.Path = append(templateVars.Path, b)
		}
		// Since the breadcrumb built is not very ordered...
		// REVERSE ALL THE THINGS
		for left, right := 0, len(templateVars.Path)-1; left < right; left, right = left+1, right-1 {
			templateVars.Path[left], templateVars.Path[right] = templateVars.Path[right], templateVars.Path[left]
		}

		// Establish list of files in the current directory
		for _, f := range files {
			templateVars.Files = append(templateVars.Files, fileItem{
				Name:  f.Name(),
				Size:  humanize.Bytes(uint64(f.Size())),
				Date:  humanize.Time(f.ModTime()),
				IsDir: f.IsDir(),
				Path:  filepath.Join(cfp, filepath.Clean(f.Name())),
			})
		}

		// Prepare the template
		tmpl, err := template.ParseFS(web.Index, "index.html")
		if err != nil {
			// Log the detailed error
			log.Println(err.Error())
			// Return a generic "Internal Server Error" message
			http.Error(w, http.StatusText(500), 500)
			return
		}

		// Return file listing in the template
		if err := tmpl.ExecuteTemplate(w, "index.html", templateVars); err != nil {
			log.Println(err.Error())
			http.Error(w, http.StatusText(500), 500)
		}
		log.Printf("200 - DIR %s\n", "/")
		return
	}

	if !info.IsDir() {
		http.ServeFile(w, r, fp)
		log.Printf("200 - FILE %s\n", cfp)
		return
	}
}
