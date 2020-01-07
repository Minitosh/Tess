package gmail

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "tokenGmail.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
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
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// GetFromGmail récupère les fiches de paie depuis Gmail
func GetFromGmail() {
	b, err := ioutil.ReadFile("credentials/gmail.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := gmail.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	user := "me"

	// Utilisateurs
	userProfile, err := srv.Users.GetProfile(user).Do()
	fmt.Printf("\nUser : %s\n", userProfile.EmailAddress)

	// Labels
	labelList, err := srv.Users.Labels.List(user).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve labels: %v", err)
	}
	if len(labelList.Labels) == 0 {
		fmt.Println("No labels found.")
		return
	}

	// Mails
	messages, err := srv.Users.Messages.List(user).Q("salaire").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve mails: %v", err)
	}
	if len(messages.Messages) == 0 {
		fmt.Println("No Messages found.")
		return
	}

	fmt.Println("\nPieces Jointes :")
	for _, message := range messages.Messages {
		messageFromID, err := srv.Users.Messages.Get(user, message.Id).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve mail from id : %v", err)
		}
		for _, part := range messageFromID.Payload.Parts {
			if len(part.Filename) > 0 {
				// Si le nom de fichier contient ("Salaire", "Paie", ect...)
				if strings.Contains(part.Filename, "salaire") || strings.Contains(part.Filename, "Salaire") || strings.Contains(part.Filename, "BDS") || strings.Contains(part.Filename, "DUE") || strings.Contains(part.Filename, "paie") {
					attachID := part.Body.AttachmentId
					// Pièce Jointe
					piece, err := srv.Users.Messages.Attachments.Get(user, message.Id, attachID).Do()
					if err != nil {
						log.Fatalf("Unable to retrieve the piece : %v", err)
					}
					fmt.Printf(" - \"%s\" : %d octets\n", part.Filename, piece.Size)
					f, err := os.Create("tmp/" + part.Filename)
					if err != nil {
						fmt.Println(err)
						return
					}
					decoded, err := base64.URLEncoding.DecodeString(piece.Data)
					n2, err := f.Write(decoded)
					if err != nil {
						fmt.Println(err)
						f.Close()
						return
					}
					fmt.Println(n2, " octets écris avec succès")
				}
			}
		}
	}
}
