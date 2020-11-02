package main

import (
	"context"
	"encoding/json"
	"github.com/k8s-autoops/autoops"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"log"
	"net/http"
	"os"
	"strings"
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
	envAnnotations := strings.Split(strings.TrimSpace(os.Getenv("CFG_ANNOTATIONS")), ",")
	for _, annotationRaw := range envAnnotations {
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
		Handler: autoops.NewMutatingAdmissionHTTPHandler(
			func(ctx context.Context, request *admissionv1.AdmissionRequest, patches *[]map[string]interface{}) (err error) {
				var buf []byte
				if buf, err = request.Object.MarshalJSON(); err != nil {
					return
				}
				var ns corev1.Namespace
				if err = json.Unmarshal(buf, &ns); err != nil {
					return
				}
				if ns.Annotations == nil {
					*patches = append(*patches, map[string]interface{}{
						"op":    "replace",
						"path":  "/metadata/annotations",
						"value": map[string]interface{}{},
					})
				}

				for k, v := range annotations {
					pk := strings.ReplaceAll(strings.ReplaceAll(k, "~", "~0"), "/", "~1")
					*patches = append(*patches, map[string]interface{}{
						"op":    "replace",
						"path":  "/metadata/annotations/" + pk,
						"value": v,
					})
				}
				return
			},
		),
	}

	if err = autoops.RunAdmissionServer(s); err != nil {
		return
	}
}
