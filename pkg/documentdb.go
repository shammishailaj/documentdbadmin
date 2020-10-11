package pkg

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"io/ioutil"
	"crypto/tls"
	"crypto/x509"
	"errors"
)

const (
	// Path to the AWS CA file
	caFilePath = "rds-combined-ca-bundle.pem"

	// Timeout operations after N seconds
	connectTimeout  = 5
	queryTimeout    = 30
	username        = "<sample-user>"
	password        = "<password>"
	clusterEndpoint = "sample-cluster.node.us-east-1.docdb.amazonaws.com:27017"

	// Which instances to read from
	readPreference = "secondaryPreferred"

	connectionStringTemplate = "mongodb://%s:%s@%s/sample-database?ssl=true&replicaSet=rs0&readpreference=%s"
)

func Connect() {

	connectionURI := fmt.Sprintf(connectionStringTemplate, username, password, clusterEndpoint, readPreference)

	tlsConfig, err := getCustomTLSConfig(caFilePath)
	if err != nil {
		log.Fatalf("Failed getting TLS configuration: %v", err)
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(connectionURI).SetTLSConfig(tlsConfig))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to cluster: %v", err)
	}

	// Force a connection to verify our connection string
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to ping cluster: %v", err)
	}

	fmt.Println("Connected to DocumentDB!")

	collection := client.Database("sample-database").Collection("sample-collection")

	ctx, cancel = context.WithTimeout(context.Background(), queryTimeout*time.Second)
	defer cancel()

	res, err := collection.InsertOne(ctx, bson.M{"name": "pi", "value": 3.14159})
	if err != nil {
		log.Fatalf("Failed to insert document: %v", err)
	}

	id := res.InsertedID
	log.Printf("Inserted document ID: %s", id)

	ctx, cancel = context.WithTimeout(context.Background(), queryTimeout*time.Second)
	defer cancel()

	cur, err := collection.Find(ctx, bson.D{})

	if err != nil {
		log.Fatalf("Failed to run find query: %v", err)
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var result bson.M
		err := cur.Decode(&result)
		log.Printf("Returned: %v", result)

		if err != nil {
			log.Fatal(err)
		}
	}

	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}

}

func getCustomTLSConfig(caFile string) (*tls.Config, error) {
	tlsConfig := new(tls.Config)
	certs, err := ioutil.ReadFile(caFile)

	if err != nil {
		return tlsConfig, err
	}

	tlsConfig.RootCAs = x509.NewCertPool()
	ok := tlsConfig.RootCAs.AppendCertsFromPEM(certs)

	if !ok {
		return tlsConfig, errors.New("Failed parsing pem file")
	}

	return tlsConfig, nil
}

