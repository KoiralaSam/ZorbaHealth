package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	healthpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_records"
)

func main() {
	bundlePath := flag.String("bundle", "examples/sample-fhir-data/demo-patient-bundle.json", "path to FHIR bundle JSON")
	patientID := flag.String("patient-id", "", "internal Zorba patient UUID (required)")
	source := flag.String("source", "synthetic-demo", "FHIR source_system label")
	addr := flag.String("addr", env.GetString("HEALTH_RECORDS_SERVICE_GRPC_ADDR", "localhost:50054"), "health-records-service gRPC address")
	flag.Parse()

	if *patientID == "" {
		log.Fatal("patient-id is required")
	}

	bundle, err := os.ReadFile(*bundlePath)
	if err != nil {
		log.Fatalf("read bundle: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	conn, err := grpcclient.Dial(*addr)
	if err != nil {
		log.Fatalf("dial health-records-service: %v", err)
	}
	defer conn.Close()

	client := healthpb.NewHealthRecordServiceClient(conn)
	resp, err := client.IngestFHIRBundle(ctx, &healthpb.FHIRBundleRequest{
		PatientId:  *patientID,
		BundleJson: string(bundle),
		Source:     *source,
	})
	if err != nil {
		log.Fatalf("ingest bundle: %v", err)
	}

	log.Printf("ingested resources=%d chunks=%d status=%s", resp.GetResourcesStored(), resp.GetChunksStored(), resp.GetStatus())
}
