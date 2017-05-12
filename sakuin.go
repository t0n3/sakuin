package main

import (
  "flag"
  "fmt"
  "github.com/dustin/go-humanize"
  "html/template"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "path/filepath"
  "strings"
)

type fileItem struct {
  Name  string
  Size  string
  Date  string
  IsDir bool
  Path  string
}

type templateVariables struct {
  Path []breadcrumb
  Files []fileItem
}

type breadcrumb struct {
  Name string
  Path string
}

var dataDir string

func main() {
  dirArg := flag.String("dir", ".", "Path to data dir you want to expose")
  portArg := flag.Int("port", 3000, "Port binded by Sakuin")
  flag.Parse()

  port := fmt.Sprintf(":%d", *portArg)

  tmpDir, err := filepath.Abs(*dirArg)
  if err != nil {
    log.Println(err)
    return
  }
  dataDir = tmpDir

  log.Println(fmt.Sprintf("Sakuin will now expose %s", dataDir))

  fs := http.FileServer(http.Dir("assets/static"))
  http.Handle("/static/", http.StripPrefix("/static/", fs))
  http.HandleFunc("/", serve)

  log.Println(fmt.Sprintf("Listening on port %s...", port))
  http.ListenAndServe(port, nil)
}

func serve(w http.ResponseWriter, r *http.Request) {
  lp := filepath.Join("assets/templates", "layout.html")
  // Filepath, from the root data dir
  fp := filepath.Join(dataDir, filepath.Clean(r.URL.Path))
  // Cleaned filepath, without the root data dir, used for template rendering purpose
  cfp := strings.Replace(fp, dataDir, "", 1)

  // Return a 404 if the template doesn't exist
  info, err := os.Stat(fp)
  if err != nil {
    if os.IsNotExist(err) {
      http.NotFound(w, r)
      log.Println(fmt.Sprintf("404 - %s", cfp))
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
    tplVars := templateVariables{}

    // Construct the breadcrumb
    path := strings.Split(cfp, "/")
    for len(path) > 0 {
      b := breadcrumb{
        Name: path[len(path)-1],
        Path: strings.Join(path, "/"),
      }
      path = path[:len(path)-1]
      tplVars.Path = append(tplVars.Path, b)
    }
    // Since the breadcrumb built is not very ordered...
    // REVERSE ALL THE THINGS
    for left, right := 0, len(tplVars.Path)-1; left < right; left, right = left+1, right-1 {
    	tplVars.Path[left], tplVars.Path[right] = tplVars.Path[right], tplVars.Path[left]
    }

    // Establish list of files in the current directory
    for _, f := range files {
      tplVars.Files = append(tplVars.Files, fileItem{
        Name: f.Name(),
        Size: humanize.Bytes(uint64(f.Size())),
        Date: humanize.Time(f.ModTime()),
        IsDir: f.IsDir(),
        Path: filepath.Join(cfp, filepath.Clean(f.Name())),
      })
    }

    // Prepare the template
    tmpl, err := template.ParseFiles(lp)
    if err != nil {
      // Log the detailed error
      log.Println(err.Error())
      // Return a generic "Internal Server Error" message
      http.Error(w, http.StatusText(500), 500)
      return
    }

    // Return file listing in the template
    if err := tmpl.ExecuteTemplate(w, "layout", tplVars); err != nil {
      log.Println(err.Error())
      http.Error(w, http.StatusText(500), 500)
    }
    log.Println(fmt.Sprintf("200 - DIR %s", cfp))
    return
  }

  if ! info.IsDir() {
    http.ServeFile(w, r, fp)
    log.Println(fmt.Sprintf("200 - FILE %s", cfp))
    return
  }
}
