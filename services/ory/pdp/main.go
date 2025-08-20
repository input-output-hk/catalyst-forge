package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type CheckIn struct {
	Namespace string `json:"namespace"`
	Object    string `json:"object"`
	Relation  string `json:"relation"`
	Subject   string `json:"subject"`
}

type ketoCheckReq struct {
	Namespace string `json:"namespace"`
	Object    string `json:"object"`
	Relation  string `json:"relation"`
	// We send simple subject_id; you could also support subject_set here.
	SubjectID string `json:"subject_id,omitempty"`
}

type ketoCheckRes struct {
	Allowed bool `json:"allowed"`
}

func main() {
	addr := getEnv("PDP_LISTEN", ":4457")
	keto := getEnv("KETO_READ_URL", "http://keto-read:4466")

	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		var in CheckIn
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		req := ketoCheckReq{
			Namespace: in.Namespace,
			Object:    in.Object,
			Relation:  in.Relation,
			SubjectID: in.Subject,
		}

		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(&req); err != nil {
			http.Error(w, "encode error", http.StatusInternalServerError)
			return
		}

		resp, err := http.Post(keto+"/relation-tuples/check", "application/json", &buf)
		if err != nil {
			log.Printf("keto error: %v", err)
			http.Error(w, "pdp upstream error", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			http.Error(w, "pdp upstream error", http.StatusBadGateway)
			return
		}

		var out ketoCheckRes
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			http.Error(w, "decode error", http.StatusBadGateway)
			return
		}

		if out.Allowed {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		} else {
			http.Error(w, "forbidden", http.StatusForbidden)
		}
	})

	log.Printf("PDP listening on %s, KETO_READ_URL=%s", addr, keto)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
