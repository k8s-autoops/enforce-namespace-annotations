package main

import (
	"context"
	"encoding/json"
	admissionv1 "k8s.io/api/admission/v1"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

type M map[string]interface{}

type Namespace struct {
	Metadata struct {
		Annotations *M `json:"annotations"`
	} `json:"metadata"`
}

const (
	certFile = "/autoops-data/tls/tls.crt"
	keyFile  = "/autoops-data/tls/tls.key"
)

func exit(err *error) {
	if *err != nil {
		log.Println("exited with error:", (*err).Error())
		os.Exit(1)
	} else {
		log.Println("exited")
	}
}

func main() {
	var err error
	defer exit(&err)

	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	annotations := map[string]string{}
	annotationRaws := strings.Split(strings.TrimSpace(os.Getenv("CFG_ANNOTATIONS")), ",")
	for _, annotationRaw := range annotationRaws {
		kv := strings.SplitN(strings.TrimSpace(annotationRaw), ":", 2)
		if len(kv) != 2 {
			continue
		}
		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		if k == "" {
			continue
		}
		annotations[k] = v
	}

	s := &http.Server{
		Addr: ":443",
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// decode request
			var review admissionv1.AdmissionReview
			if err := json.NewDecoder(req.Body).Decode(&review); err != nil {
				log.Println("Failed to decode a AdmissionReview:", err.Error())
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}

			// log
			reviewPrettyJSON, _ := json.MarshalIndent(&review, "", "  ")
			log.Println(string(reviewPrettyJSON))

			// patches
			var buf []byte
			var ns Namespace

			if buf, err = review.Request.Object.MarshalJSON(); err != nil {
				log.Println("Failed to marshal object to json:", err.Error())
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}
			if err = json.Unmarshal(buf, &ns); err != nil {
				log.Println("Failed to unmarshal object to statefulset:", err.Error())
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}

			// build patches
			var patches []M
			if ns.Metadata.Annotations == nil {
				patches = append(patches, M{
					"op":    "replace",
					"path":  "/metadata/annotations",
					"value": M{},
				})
			}

			for k, v := range annotations {
				pk := strings.ReplaceAll(strings.ReplaceAll(k, "~", "~0"), "/", "~1")
				patches = append(patches, M{
					"op":    "replace",
					"path":  "/metadata/annotations/" + pk,
					"value": v,
				})
			}

			// build response
			var patchJSON []byte
			if patchJSON, err = json.Marshal(patches); err != nil {
				log.Println("Failed to marshal patches:", err.Error())
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}
			patchType := admissionv1.PatchTypeJSONPatch
			review.Response = &admissionv1.AdmissionResponse{
				UID:       review.Request.UID,
				Allowed:   true,
				Patch:     patchJSON,
				PatchType: &patchType,
			}
			review.Request = nil

			// send response
			reviewJSON, _ := json.Marshal(review)
			rw.Header().Set("Content-Type", "application/json")
			rw.Header().Set("Content-Length", strconv.Itoa(len(reviewJSON)))
			_, _ = rw.Write(reviewJSON)
		}),
	}

	// channels
	chErr := make(chan error, 1)
	chSig := make(chan os.Signal, 1)
	signal.Notify(chSig, syscall.SIGTERM, syscall.SIGINT)

	// start server
	go func() {
		log.Println("listening at :443")
		chErr <- s.ListenAndServeTLS(certFile, keyFile)
	}()

	// wait signal or failed start
	select {
	case err = <-chErr:
	case sig := <-chSig:
		log.Println("signal caught:", sig.String())
		_ = s.Shutdown(context.Background())
	}
}
