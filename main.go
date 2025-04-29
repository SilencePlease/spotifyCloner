package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

const redirectURI = "http://127.0.0.1:8080/callback"

var (
	auth = spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserReadPrivate,
			spotifyauth.ScopePlaylistReadPrivate,
			spotifyauth.ScopePlaylistReadCollaborative,
			spotifyauth.ScopePlaylistModifyPublic,
			spotifyauth.ScopePlaylistModifyPrivate,
		),
	)
	ch    = make(chan *spotify.Client)
	state = "abc123"
	// Auth for account everything is meant to be coppied to
	authAccTwo = spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserReadPrivate,
			spotifyauth.ScopePlaylistReadPrivate,
			spotifyauth.ScopePlaylistReadCollaborative,
			spotifyauth.ScopePlaylistModifyPublic,
			spotifyauth.ScopePlaylistModifyPrivate,
		),
	)
	chTwo    = make(chan *spotify.Client)
	stateTwo = "abc124"
)

func main() {
	// Start HTTP Server für OAuth Callback
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Authentifizierungs-URL ausgeben
	url := auth.AuthURL(state)
	fmt.Println("Bitte öffne die folgende URL im Browser und logge dich bei Spotify ein:", url)

	// Warten, bis Authentifizierung abgeschlossen ist
	client := <-ch

	// Aktuellen Benutzer abrufen
	user, err := client.CurrentUser(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Eingeloggt als:", user.ID)

	// Playlists des Benutzers abrufen
	playlists, err := client.CurrentUsersPlaylists(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	personalPlaylists := make(map[string][]string)
	nonPersonalPlaylists := make(map[string][]string)

	fmt.Println("\nDeine Playlists:")
	for _, playlist := range playlists.Playlists {
		fmt.Printf("• %s (ID: %s)\n", playlist.Name, playlist.ID)

		if playlist.Owner.ID == user.ID {
			// Playlist-Items (Songs) abrufen
			items, err := client.GetPlaylistItems(context.Background(), playlist.ID)
			if err != nil {
				log.Fatal(err)
			}

			// Slice für Song-IDs initialisieren
			var personalSongIDs []string

			// Alle Songs durchgehen
			for _, item := range items.Items {
				if item.Track.Track != nil { // zur Sicherheit prüfen, ob Track vorhanden ist
					personalSongIDs = append(personalSongIDs, string(item.Track.Track.ID))
				}
			}

			// In Map speichern: Playlistname → Song-IDs
			personalPlaylists[playlist.Name] = personalSongIDs
		} else {
			// Playlist-Items (Songs) abrufen
			items, err := client.GetPlaylistItems(context.Background(), playlist.ID)
			if err != nil {
				log.Fatal(err)
			}

			// Slice für Song-IDs initialisieren
			var nonPersonalSongIDs []string

			// Alle Songs durchgehen
			for _, item := range items.Items {
				if item.Track.Track != nil { // zur Sicherheit prüfen, ob Track vorhanden ist
					nonPersonalSongIDs = append(nonPersonalSongIDs, string(item.Track.Track.ID))
				}
			}

			// In Map speichern: Playlistname → Song-IDs
			nonPersonalPlaylists[playlist.Name] = nonPersonalSongIDs
		}
	}
	fmt.Println("\nDeine Personal Playlists:", personalPlaylists)

}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Konnte Token nicht abrufen", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}

	// Authentifizierten Spotify-Client erstellen
	client := spotify.New(auth.Client(r.Context(), tok))
	fmt.Fprint(w, "Login erfolgreich. Du kannst das Terminal weiterverwenden.")
	ch <- client
}
