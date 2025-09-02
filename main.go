package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"time"

	"github.com/mikelu92/emailimport/provider"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v2"
)

const user = "me"

type Config struct {
	Providers       []provider.ProviderConfig `yaml:"providers"`
	Processed       string                    `yaml:"processedLabel"`
	CredentialsFile string                    `yaml:"credentials"`
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = tokenFromWeb(context.Background(), config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func tokenFromWeb(ctx context.Context, config *oauth2.Config) *oauth2.Token {
	ch := make(chan string)
	randState := fmt.Sprintf("st%d", time.Now().UnixNano())
	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/favicon.ico" {
			http.Error(rw, "", 404)
			return
		}
		if req.FormValue("state") != randState {
			log.Printf("State doesn't match: req = %#v", req)
			http.Error(rw, "", 500)
			return
		}
		if code := req.FormValue("code"); code != "" {
			fmt.Fprintf(rw, "<h1>Success</h1>Authorized.")
			rw.(http.Flusher).Flush()
			ch <- code
			return
		}
		log.Printf("no code")
		http.Error(rw, "", 500)
	}))
	defer ts.Close()

	// Ensure redirect URI uses localhost instead of 127.0.0.1 so it matches the
	// allowed URIs in Google Cloud Console, which typically include only
	// "http://localhost". We keep the dynamic port chosen by httptest.
	u, _ := url.Parse(ts.URL)
	config.RedirectURL = fmt.Sprintf("http://localhost:%s", u.Port())
	// Request offline access so we receive a refresh token that allows the app to
	// reuse the credentials without re-authorisation.
	authURL := config.AuthCodeURL(randState, oauth2.AccessTypeOffline)
	go openURL(authURL)
	log.Printf("Authorize this app at: %s", authURL)
	code := <-ch
	log.Printf("Got code: %s", code)

	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Fatalf("Token exchange error: %v", err)
	}
	return token
}

func openURL(url string) {
	try := []string{"xdg-open", "google-chrome", "open"}
	for _, bin := range try {
		err := exec.Command(bin, url).Run()
		if err == nil {
			return
		}
	}
	log.Printf("Error opening URL in browser.")
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func main() {
	ctx := context.Background()

	flag.String("config", "config.yaml", "config for providers")
	var c Config
	err := ReadConfig("config.yaml", &c)
	if err != nil {
		log.Fatalf("Could not get config file")
	}
	b, err := os.ReadFile(c.CredentialsFile)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}
	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope, gmail.GmailModifyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}
	args := os.Args[1:]
	if len(args) > 0 {
		if args[0] == "labels" {
			showLabels(srv)
		} else if args[0] == "threads" {
			getThreads(srv)
		}
		return
	}

	r, err := srv.Users.Messages.List(user).LabelIds("UNREAD").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve messages: %v", err)
	}
	if len(r.Messages) == 0 {
		log.Fatalf("No messages found.\n")
		return
	}

	for i := len(r.Messages) - 1; i >= 0; i-- {
		m := r.Messages[i]
		msg, err := srv.Users.Messages.Get(user, m.Id).Do()
		if err != nil {
			log.Fatalf("couldn't get msg %q\n", m.Id)
			return
		}
		var p provider.Provider
	find:
		for _, id := range msg.LabelIds {
			for _, pr := range c.Providers {
				if id == pr.Label {
					p = provider.Get(pr)
					break find
				}
			}
		}
		if p == nil {
			continue
		}
		t, err := p.GetTransaction(msg)
		if err != nil {
			log.Fatalf("unable to get transaction from email: %v", err)
		}
		if t == nil {
			log.Printf("unrecognized transaction format for account %q, but will continue\n", p.GetAccount())
			continue
		}

		fmt.Print(t.Print())
		_, err = srv.Users.Messages.Modify(user, m.Id, &gmail.ModifyMessageRequest{AddLabelIds: []string{c.Processed}, RemoveLabelIds: []string{"UNREAD", "INBOX"}}).Do()
		if err != nil {
			log.Fatalf("couldn't modify message %q", m.Id)
		}
	}

	// Now get any threads that have unread messages
	rt, err := srv.Users.Threads.List(user).LabelIds("UNREAD").Do()
	if err != nil {
		log.Fatalf("couldn't get threads")
	}
	for i := len(rt.Threads) - 1; i >= 0; i-- {
		t := rt.Threads[i]
		ths, err := srv.Users.Threads.Get(user, t.Id).Do()
		if err != nil {
			log.Fatalf("could not get messages from thread")
		}
		var p provider.Provider
		// only the first message in the thread will have our provider label ids
	findTh:
		for _, l := range ths.Messages[0].LabelIds {
			for _, pr := range c.Providers {
				if l == pr.Label {
					p = provider.Get(pr)
					break findTh
				}
			}
		}
		if p == nil {
			continue
		}
		slices.Reverse(ths.Messages)
		for _, m := range ths.Messages {
			var unread bool
			for _, l := range m.LabelIds {
				if l == "UNREAD" {
					unread = true
					break
				}
			}
			if !unread {
				continue
			}
			t, err := p.GetTransaction(m)
			if err != nil {
				log.Fatalf("could not get transaction from thread message")
			}
			if t == nil {
				log.Printf("unrecognized transaction format for account %q, but will continue\n", p.GetAccount())
				continue
			}

			fmt.Print(t.Print())

			_, err = srv.Users.Messages.Modify(user, m.Id, &gmail.ModifyMessageRequest{AddLabelIds: []string{c.Processed}, RemoveLabelIds: []string{"UNREAD", "INBOX"}}).Do()
			if err != nil {
				log.Fatalf("couldn't modify message %q", m.Id)
			}
		}
	}
}

func getThreads(srv *gmail.Service) {
	r, err := srv.Users.Threads.List(user).LabelIds("Label_1454095201736435186").Do()
	if err != nil {
		log.Fatalf("couldn't get threads")
	}
	for _, t := range r.Threads {
		th, _ := srv.Users.Threads.Get(user, t.Id).Do()
		for _, m := range th.Messages {
			fmt.Printf("%+v\n", m)
		}
	}
}

func showLabels(srv *gmail.Service) {
	r, err := srv.Users.Labels.List(user).Do()
	if err != nil {
		log.Fatalf("Could not get labels")
	}
	for _, l := range r.Labels {
		fmt.Printf("%s: %s\n", l.Id, l.Name)
	}
}

func ReadConfig(path string, config interface{}) error {
	if path == "" {
		return fmt.Errorf("No config path provided, please supply a value for -config")
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("Failed to open config file %s: %v", path, err)
	}
	defer f.Close()

	confBuf, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("Failed to read config file %s: %v", path, err)
	}

	err = yaml.Unmarshal(confBuf, config)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal config file %s: %v", path, err)
	}
	return nil
}
