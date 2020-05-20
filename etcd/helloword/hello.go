package main

import (
	"context"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/sirupsen/logrus"
	"i-go/core/etcd"
	"strconv"
	"time"
)

var (
	client = etcd.CliV3
	kv     = clientv3.NewKV(client)
	lease  = clientv3.NewLease(client)
)

const (
	Prefix = "/hello"
	Suffix = "/2"
)

func main() {
	defer etcd.Release()

	// put()
	// get()
	// delete()
	//leaseFunc()
	txn()
	// watch机制
	// go putForWatch()
	// go watch()
	// select {}
}
func putForWatch() {
	for i := 0; i < 9; i++ {
		_, err := kv.Put(context.Background(), Prefix+Suffix, strconv.Itoa(i))
		if err != nil {
			logrus.WithFields(logrus.Fields{"Scenes": "etcd put"}).Error(err)
		}
		time.Sleep(time.Millisecond * 100)
	}
	_, err := kv.Delete(context.Background(), Prefix+Suffix)
	if err != nil {
		logrus.WithFields(logrus.Fields{"Scenes": "etcd put"}).Error(err)
	}
}
func watch() {
	watchChan := client.Watch(context.Background(), Prefix+Suffix)
	for wr := range watchChan {
		for _, e := range wr.Events {
			switch e.Type {
			case clientv3.EventTypePut:
				fmt.Printf("watch event put-current: %#v \n", string(e.Kv.Value))
			case clientv3.EventTypeDelete:
				fmt.Printf("watch event delete-current: %#v \n", string(e.Kv.Value))
			default:
			}
		}
	}
}

func txn() {
	kv.Put(context.Background(), Prefix+Suffix, "f")
	// 开启事务
	txn := kv.Txn(context.Background())
	getOwner := clientv3.OpGet(Prefix+Suffix, clientv3.WithFirstCreate()...)
	// 如果/illusory/cloud的值为hello则获取/illusory/cloud的值 否则获取/illusory/wind的值
	txnResp, err := txn.If(clientv3.Compare(clientv3.Value(Prefix+Suffix), "=", "hello")).
		Then(clientv3.OpGet(Prefix+"/equal"), getOwner).
		Else(clientv3.OpGet(Prefix+"/unequal"), getOwner).
		Commit()
	if err != nil {
		logrus.WithFields(logrus.Fields{"Scenes": "etcd put"}).Error(err)
		return
	}
	fmt.Printf("事务%#v \n", txnResp)
	if txnResp.Succeeded { // If = true
		fmt.Println("true", txnResp.Responses[0].GetResponseRange().Kvs)
	} else { // If =false
		fmt.Println("false", txnResp.Responses[0].GetResponseRange().Kvs)
	}
	kv.Delete(context.Background(), Prefix+Suffix)
}

func put() {
	response, err := kv.Put(context.Background(), Prefix+Suffix, "hello")
	response, err = kv.Put(context.Background(), Prefix+"/equal", "equal")
	response, err = kv.Put(context.Background(), Prefix+"/unequal", "unequal")
	if err != nil {
		logrus.WithFields(logrus.Fields{"Scenes": "etcd put"}).Error(err)
	}
	fmt.Println(response)
}

func get() {
	response, err := kv.Get(context.Background(), Prefix+Suffix)
	if err != nil {
		logrus.WithFields(logrus.Fields{"Scenes": "etcd get"}).Error(err)
	}
	fmt.Println(response)
}

func delete() {
	response, err := kv.Delete(context.Background(), Prefix+Suffix)
	if err != nil {
		logrus.WithFields(logrus.Fields{"Scenes": "etcd delete"}).Error(err)
	}
	fmt.Println(response)
}

// leaseFunc lease租约
func leaseFunc() {
	response, err := lease.Grant(context.Background(), 10)
	if err != nil {
		logrus.WithFields(logrus.Fields{"Scenes": "etcd delete"}).Error(err)
	}
	leaseID := response.ID
	fmt.Printf("leaseID:%v TTL:%v \n", leaseID, response.TTL)
	_, err = kv.Put(context.Background(), Prefix+"/lease1", "lease1", clientv3.WithLease(leaseID))
	if err != nil {
		logrus.WithFields(logrus.Fields{"Scenes": "etcd Put"}).Error(err)
	}
	leasesResponse, err := lease.Leases(context.Background())
	if err != nil {
		logrus.WithFields(logrus.Fields{"Scenes": "etcd Leases"}).Error(err)
	}
	fmt.Printf("Leases:%v \n", leasesResponse.Leases)
	// 主动给Lease进行续约
	keepAliveChan, err := client.KeepAlive(context.TODO(), leaseID)
	if err != nil { // 有协程来帮自动续租,TTL剩余一半时就会续约。
		fmt.Println(err)
		return
	} else {
		go func() {
			for {
				select {
				case resp := <-keepAliveChan:
					fmt.Println("续租:", resp)
					// if resp == nil {
					// 	fmt.Println("续租失败")
					// 	break
					// } else {
					// 	fmt.Println("续租成功")
					// }
				}
				break
			}
		}()
	}
	for {
		time.Sleep(time.Millisecond * 500)
		liveResponse, err := lease.TimeToLive(context.Background(), leaseID)
		if err != nil {
			logrus.WithFields(logrus.Fields{"Scenes": "etcd TimeToLive"}).Error(err)
		}
		fmt.Printf("leaseID:%v TTL:%v GrantedTTL:%v Keys:%v \n", liveResponse.ID, liveResponse.TTL, liveResponse.GrantedTTL, liveResponse.Keys)
		if liveResponse.TTL == -1 {
			break
		}
	}
}