package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"

	"github.com/kelseyhightower/envconfig"
	"github.com/srvaroa/repocket/pkg/pocket"
	"github.com/srvaroa/repocket/pkg/repocket"
)

func ensureDir(path string) {
	f, err := os.Stat(path)
	if os.IsNotExist(err) {
		if err != nil {
			log.Fatalf("ERROR: expecting directory %s to exist", path)
		}
	}
	if !f.IsDir() {
		log.Fatalf("ERROR: expecting path %s to be a directory", path)
	}
}

func getArticleContents(a *pocket.Article) ([]byte, error) {
	return exec.Command("w3m", "-dump", a.ResolvedUrl).Output()
}

func dumpArticle(outputDir string, a *pocket.Article) {

	// Clean up path
	re := regexp.MustCompile(`[\.|/\\]+`)
	path := outputDir + "/" +
		a.ItemId +
		"_" +
		string(re.ReplaceAll([]byte(a.ResolvedTitle), []byte("-")))

	// If the article is there, leave it alone
	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		log.Printf("Skipping (already downloaded): %s", a.ResolvedTitle)
		return
	}

	log.Printf("Downloading: `%s` to `%s`", a.ResolvedTitle, path)

	out, err := getArticleContents(a)
	if err != nil {
		log.Print("Failed to download %s, %s", a.ResolvedUrl, err)
		return
	}

	file, err := os.Create(path)
	if err != nil {
		log.Printf("Failed to create file for %s: %s", a.ResolvedTitle, err)
		return
	}
	defer file.Close()

	_, err = io.WriteString(file, string(out))
	if err != nil {
		log.Printf("Failed to write %s: %s", a.ResolvedTitle, err)
	}
}

func makeConfig() repocket.Config {
	cfg := repocket.Config{}
	err := envconfig.Process("REPOCKET", &cfg)
	if err != nil {
		log.Fatal(err.Error())
	}
	return cfg
}

func authenticate(cfg *repocket.Config) {
	var err error
	cfg.AccessToken, err = repocket.LoadLocalConfig()
	if err != nil {
		log.Printf("Could not load local config, authorizing against"+
			" GetPocket's API: %s", err)
		cfg.AccessToken, err = pocket.Authorize(cfg.ConsumerKey)
		if err != nil {
			log.Fatal("Failed to authorize against Pocket: %s", err)
		}
		err = repocket.SaveLocalConfig(cfg.AccessToken)
		if err != nil {
			log.Printf("Failed to persist user token: %s", err)
		}
	}
}

// dump reads all articles marked as favourite and dumps them in the
// desired output directory
func dump(cfg repocket.Config) {
	favs := pocket.QueryFavourites(cfg.AccessToken, cfg.ConsumerKey)
	if len(cfg.OutputDir) == 0 {
		log.Fatalf("No output directory provided " +
			"(expected at the REPOCKET_OUTPUT_DIR env variable)")
	}
	ensureDir(cfg.OutputDir)
	for _, item := range favs {
		dumpArticle(cfg.OutputDir, &item)
	}
}

// list all the starred articles
func list(cfg repocket.Config) {
	favs := pocket.QueryFavourites(cfg.AccessToken, cfg.ConsumerKey)
	for _, item := range favs {
		log.Println(fmt.Sprintf("| %-50.50s | %s", item.ResolvedTitle, item.ResolvedUrl))
	}
}

// next dumps the next unread article
func next(cfg repocket.Config) {
	favs := pocket.QueryNewest(cfg.AccessToken, cfg.ConsumerKey, 1)
	for _, a := range favs {
		out, err := getArticleContents(&a)
		if err != nil {
			log.Fatalf("Failed to get article contents: %s", err)
		}
		log.Printf("%s", out)
	}
}

type logWriter struct{}

func usageAndExit() {
	log.Fatal("Usage: %s [dump|list]\n", os.Args[0])
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(string(bytes))
}

func main() {

	log.SetFlags(0)
	log.SetOutput(new(logWriter))

	if len(os.Args) != 2 {
		usageAndExit()
	}

	cmd := os.Args[1]

	cfg := makeConfig()
	authenticate(&cfg)

	switch cmd {
	case "dump":
		dump(cfg)
		break
	case "list":
		list(cfg)
		break
	case "next":
		next(cfg)
		break
	default:
		usageAndExit()
	}

}
