package main

import (
	"encoding/json"
	"log"
)

func DecodeSerializedData(data any, target any) error {
	bs, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(bs, target)
}

func EncodeSerializedData(data any) string {
	bs, err := json.Marshal(data)
	if err != nil {
		log.Printf("序列化数据失败: %v", err)
		return "序列化数据失败"
	}
	return string(bs)
}
