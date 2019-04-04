package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Request struct {
	Id string `json:"id"`
	Method string `json:"method"`
	Parameters map[string]string `json:"parameters"`
	Response Response `json:"response"`
}

type RequestsList struct {
	Requests []Request `json:"requests"`
}

type Response struct {
	RequestId string `json:"request_id"`
	Result interface{} `json:"result"`
}

type Executor struct {
	Host string `json:"host"`
	Port int `json:"port"`
	Weight int `json:"weight"`
	LastUsage time.Time `json:"last_usage"`

	Statistics []ExecutorStatistics `json:"statistics"`
	AverageResponseTime float64 `json:"average_response_time"`
	ErrorsCount int `json:"errors_count"`
	
	Blocked bool `json:"blocked"`
	BlockReason string `json:"block_reason"`
}

func (executor *Executor) Block(reason string) {
	executor.Blocked = true
	executor.BlockReason = reason
}

type ExecutorStatistics struct {
	Timeline float64 `json:"timeline"`
}

var ExecutorList []Executor
var RegisteredRequestList []RequestsList
var ProcessedRequestsCount int
var regHash string

func getFreeExecutor() (*Executor, error) {
	var freeExecutors = map[string]Executor{}

	for _, exec := range ExecutorList {
		//Executor free where it never used or used greater then 1 seconds ago
		if !exec.Blocked && (exec.LastUsage.IsZero() || time.Now().Sub(exec.LastUsage).Seconds() > 1) {
			freeExecutors[exec.Host] = exec
		}
	}

	if len(freeExecutors) == 0 {
		fmt.Println("No free executors... Sleep 1 second and try later")
		return nil, errors.New("No free executors")
	}

	var maxUnusedSeconds = 0.0
	var priorityExec Executor

	for _, freeExec := range freeExecutors {
		var unUsedSeconds = time.Now().Sub(freeExec.LastUsage).Seconds()
		if unUsedSeconds > maxUnusedSeconds {
			maxUnusedSeconds = unUsedSeconds
			priorityExec = freeExec
		}
	}

	for i, exec := range ExecutorList {
		if exec.Host == priorityExec.Host {
			ExecutorList[i].LastUsage = time.Now()
		}
	}

	fmt.Println("Choose " + priorityExec.Host)

	return &priorityExec, nil
}

func FindRealExecutor(host string) * Executor {
	for i, ex := range ExecutorList {
		if ex.Host == host {
			return &ExecutorList[i]
		}
	}

	return nil
}

func doRequest(request Request) interface{} {
	jsonStr, _ := json.Marshal(request)

	getFreeExecutorWrap := func() *Executor {
		var executor *Executor
		var err error
		//Wait free executor
		for true {
			// No free executors? Need min time for wait free executor
			executor, err = getFreeExecutor(); if err != nil {
				sleep := 1.0 //Max wait time

				//Search min value for wait free node
				for _, ex := range ExecutorList {
					diff := time.Now().Sub(ex.LastUsage).Seconds()
					if diff < sleep {
						sleep = diff
					}
				}

				fmt.Printf("No free executors... Sleep %f second and try later\r\n", sleep)
				time.Sleep(time.Duration(sleep) * time.Millisecond)
			} else {
				break
			}
		}

		return executor
	}

	var executor = getFreeExecutorWrap()

	var uri strings.Builder

	uri.WriteString("http://")
	uri.WriteString(executor.Host)
	uri.WriteString("/request")

	start := time.Now()
	req, err := http.NewRequest("POST", uri.String(), bytes.NewBuffer(jsonStr)); if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	end := time.Now()

	//Start store metrics about performance
	realExecutor := FindRealExecutor(executor.Host)

	if err != nil {
		log.Println("Error in " + executor.Host + " " + err.Error())

		realExecutor.ErrorsCount++
		if realExecutor.ErrorsCount >= 100 {
			//realExecutor.Block("Errors count >= 100")
		}

		return doRequest(request)
	}

	realExecutor.Statistics = append(realExecutor.Statistics, ExecutorStatistics{ Timeline: end.Sub(start).Seconds() })

	//Store last five statistic info only for optimize memory usage and fix memory leak
	if len(realExecutor.Statistics) >= 10 {
		var sum = 0.0
		for _, stat := range realExecutor.Statistics {
			sum += stat.Timeline
		}
		realExecutor.AverageResponseTime = sum / float64(len(realExecutor.Statistics))
		realExecutor.Statistics = []ExecutorStatistics{} //Empty metrics
	}

	body, _ := ioutil.ReadAll(resp.Body)
	var response interface{}
	unmarshalErr := json.Unmarshal(body, &response); if unmarshalErr != nil {
		log.Fatal(unmarshalErr)
	}

	ProcessedRequestsCount += 1

	return response
}

func handleErrResponse(w http.ResponseWriter, errDesc string, statusCode int, fatal bool) {
	w.WriteHeader(statusCode)

	_, err := w.Write([]byte(errDesc)); if err != nil {
		log.Fatal(err)
	}

	if fatal {
		log.Fatal(errDesc)
	} else {
		log.Println(errDesc)
	}

	return
}

func StatisticsHandler(w http.ResponseWriter, r *http.Request) {
	type SystemStatisticsResponse struct {
		ProcessedRequestsCount int `json:"processed_requests_count"`
		ExecutorList []Executor `json:"executor_list"`
	}

	res := SystemStatisticsResponse{}
	res.ProcessedRequestsCount = ProcessedRequestsCount
	res.ExecutorList = ExecutorList

	_ = json.NewEncoder(w).Encode(res)
}

func RegisterExecutor(w http.ResponseWriter, r *http.Request) {
	var newExecutor Executor
	_ = json.NewDecoder(r.Body).Decode(&newExecutor)

	auth := r.Header.Get("Authorization")
	if auth != regHash {
		handleErrResponse(w, "Authorization error", 401, false)
		return
	}

	req, err := http.NewRequest("GET", newExecutor.Host + "/_health", nil); if err != nil {
		handleErrResponse(w, "Internal error", 500, false)
		return
	}

	client := http.Client{}
	res, err := client.Do(req); if err != nil {
		handleErrResponse(w, "Health check error: " + err.Error(), 400, false)
		return
	}

	body, err := ioutil.ReadAll(res.Body); if err != nil {
		handleErrResponse(w, "Internal error", 500, false)
		return
	}

	if string(body) != "OK" {
		handleErrResponse(w, "Health check error: response need \"OK\" Your response" + string(body), 400, false)
		return
	}

	ExecutorList = append(ExecutorList, newExecutor)

	_, errMain := w.Write([]byte("OK")); if errMain != nil {
		log.Fatal(errMain)
	}
}

func CreateRequests(w http.ResponseWriter, r *http.Request)  {
	var requestList RequestsList
	_ = json.NewDecoder(r.Body).Decode(&requestList)
	RegisteredRequestList = append(RegisteredRequestList, requestList)

	//Responses list
	responsesChan := make(chan Response)

	//Create requests tasks and wait done it later
	for _, request := range requestList.Requests {
		go func (req Request, channelForResponse chan Response) {
			channelForResponse <- Response {
				RequestId: req.Id,
				Result: doRequest(req),
			}
		}(request, responsesChan)
	}

	//Wait && later fill responses to requests
	for i := 0; i < len(requestList.Requests); i++ {
		response := <-responsesChan
		for k, _ := range requestList.Requests {
			if requestList.Requests[k].Id == response.RequestId {
				requestList.Requests[k].Response = response
			}
		}
	}

	close(responsesChan)

	err := json.NewEncoder(w).Encode(requestList); if err != nil {
		log.Fatal(err)
	}
}

func main() {
	h := sha256.New()
	regHash = fmt.Sprintf("%x", h.Sum([]byte(string(time.Now().Unix()))))

	log.Println("Hash for authorization " + regHash)

	executors := strings.Split(os.Getenv("EXECUTORS"),",")
	for _, executor := range executors {
		log.Println("Start register executor: " + executor)
		ExecutorList = append(ExecutorList, Executor{ Host: executor })
	}

	router := mux.NewRouter()
	router.HandleFunc("/register-executor", RegisterExecutor).Methods("POST")
	router.HandleFunc("/", StatisticsHandler).Methods("GET", "POST")
	router.HandleFunc("/requests", CreateRequests).Methods("POST")
	log.Fatal(http.ListenAndServe(":8000", router))
}
