package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"

	"github.com/gorilla/mux"
)

type job struct {
	Name     string
	Duration int
}

type agent struct {
	AgentID int
}

type payload struct {
	Agents []agent
	Jobs   []job
}

type processAgent struct {
	ID       int
	Capacity int
	Jobs     []job
}

func (agent *processAgent) Add(j job) {
	agent.Capacity -= j.Duration
	agent.Jobs = append(agent.Jobs, j)
}

func (agent *processAgent) Remove(j job) {
	agent.Capacity += j.Duration

	var indexValue int

	for i := range agent.Jobs {
		if agent.Jobs[i].Name == j.Name {
			indexValue = i
		}
	}

	agent.Jobs = agent.Jobs[:indexValue+copy(agent.Jobs[indexValue:], agent.Jobs[indexValue+1:])]
}

func getHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Healthy!")
}

func max(items []job) (max int) {
	max = 0

	for _, item := range items {
		if item.Duration > max {
			max = item.Duration
		}
	}

	return max
}

func jobRemove(jobs []job, itemToRemove job) []job {
	var indexValue int

	for i := range jobs {
		if jobs[i].Name == itemToRemove.Name {
			indexValue = i
		}
	}

	return jobs[:indexValue+copy(jobs[indexValue:], jobs[indexValue+1:])]
}

func jobSort(jobs []job, agents []processAgent) []processAgent {

	jobsCopy := make([]job, len(jobs))
	copy(jobsCopy, jobs)

	for _, job := range jobsCopy {
		for i, agent := range agents {
			if job.Duration > agent.Capacity {
				continue
			}
			agents[i].Add(job)
			jobs = jobRemove(jobs, job)
			break
		}
	}

	//Agents are overcapacity so allocate best
	if len(jobs) > 0 {
		for len(jobs) > 0 {

			sort.Slice(agents, func(i, j int) bool {
				return agents[i].Capacity > agents[j].Capacity
			})

			for i := range agents {
				agents[i].Add(jobs[0])
				jobs = jobRemove(jobs, jobs[0])
				break
			}
		}
	}

	return agents
}

func process(w http.ResponseWriter, r *http.Request) {
	var body payload
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Body error")
	}

	json.Unmarshal(reqBody, &body)

	var agents []processAgent

	maxValue := max(body.Jobs)

	for _, agent := range body.Agents {
		agents = append(agents, processAgent{ID: agent.AgentID, Capacity: maxValue})
	}

	json.NewEncoder(w).Encode(jobSort(body.Jobs, agents))
}

func main() {

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/health", getHealth).Methods("GET")
	router.HandleFunc("/process", process).Methods("POST")
	log.Fatal(http.ListenAndServe(":8080", router))
}
