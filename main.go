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
	"syscall"
)

type M map[string]interface{}

type StatefulSet struct {
	Spec struct {
		Template struct {
			Metadata struct {
				Annotations *M `json:"annotations"`
			} `json:"metadata"`
			Spec struct {
				Containers []struct {
					Resources *struct {
						Limits   *M `json:"limits"`
						Requests *M `json:"requests"`
					} `json:"resources"`
				} `json:"containers"`
			} `json:"spec"`
		} `json:"template"`
	} `json:"spec"`
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
			var sts StatefulSet

			if buf, err = review.Request.Object.MarshalJSON(); err != nil {
				log.Println("Failed to marshal object to json:", err.Error())
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}
			if err = json.Unmarshal(buf, &sts); err != nil {
				log.Println("Failed to unmarshal object to statefulset:", err.Error())
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}

			// build patches
			var patches []M
			if sts.Spec.Template.Metadata.Annotations == nil {
				patches = append(patches, M{
					"op":    "replace",
					"path":  "/spec/template/metadata/annotations",
					"value": M{},
				})
			}
			patches = append(patches, M{
				"op":    "replace",
				"path":  "/spec/template/metadata/annotations/tke.cloud.tencent.com~1networks",
				"value": "tke-route-eni",
			})
			patches = append(patches, M{
				"op":    "replace",
				"path":  "/spec/template/metadata/annotations/tke.cloud.tencent.com~1vpc-ip-claim-delete-policy",
				"value": "Never",
			})
			c := sts.Spec.Template.Spec.Containers[0]
			if c.Resources == nil {
				patches = append(patches, M{
					"op":    "replace",
					"path":  "/spec/template/spec/containers/0/resources",
					"value": M{},
				})
			}
			if c.Resources == nil || c.Resources.Limits == nil {
				patches = append(patches, M{
					"op":    "replace",
					"path":  "/spec/template/spec/containers/0/resources/limits",
					"value": M{},
				})
			}
			if c.Resources == nil || c.Resources.Requests == nil {
				patches = append(patches, M{
					"op":    "replace",
					"path":  "/spec/template/spec/containers/0/resources/requests",
					"value": M{},
				})
			}
			patches = append(patches, M{
				"op":    "replace",
				"path":  "/spec/template/spec/containers/0/resources/limits/tke.cloud.tencent.com~1eni-ip",
				"value": "1",
			})
			patches = append(patches, M{
				"op":    "replace",
				"path":  "/spec/template/spec/containers/0/resources/requests/tke.cloud.tencent.com~1eni-ip",
				"value": "1",
			})

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
