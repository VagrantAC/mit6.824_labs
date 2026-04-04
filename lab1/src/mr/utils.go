package mr

import (
	"encoding/json"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

func GenReduceTaskFilename(filename string, workerId, reduceId int) string {
	return fmt.Sprintf("mr-%s-%d-%d", filename, workerId, reduceId)
}

func ReadKeyValues(filename string) ([]KeyValue, error) {
	file, err := os.Open(filename)
	if err != nil {
		log.Errorf("cannot open %v", filename)
		return nil, err
	}
	defer file.Close()

	kva := []KeyValue{}
	dec := json.NewDecoder(file)
	for {
		var kv KeyValue
		if err := dec.Decode(&kv); err != nil {
			log.Errorf("cannot decode %v", err)
			break
		}
		kva = append(kva, kv)
	}
	return kva, nil
}

func WriteKeyValues(filename string, kva []KeyValue) error {
	file, err := os.Create(filename)
	if err != nil {
		log.Errorf("cannot create %v", filename)
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	for _, kv := range kva {
		if err := enc.Encode(kv); err != nil {
			log.Errorf("cannot encode %v", kv)
			return err
		}
	}
	return nil
}
