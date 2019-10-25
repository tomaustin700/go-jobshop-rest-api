package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"sort"

	"github.com/gorilla/mux"
)

type job struct {
	Name     string
	Duration int
}

type resource struct {
	ResourceID int
}

type payload struct {
	Resources []resource
	Jobs      []job
}

type processResource struct {
	ID       int
	Capacity int
	Jobs     []job
}

func (resource *processResource) Add(j job) {
	resource.Capacity -= j.Duration
	resource.Jobs = append(resource.Jobs, j)
}

func (resource *processResource) Remove(j job) {
	resource.Capacity += j.Duration

	var indexValue int

	for i := range resource.Jobs {
		if resource.Jobs[i].Name == j.Name {
			indexValue = i
		}
	}

	resource.Jobs = resource.Jobs[:indexValue+copy(resource.Jobs[indexValue:], resource.Jobs[indexValue+1:])]
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

func anyEmptyResources(vs []processResource) bool {
	for _, v := range vs {
		if len(v.Jobs) == 0 {
			return true
		}
	}
	return false
}

func average(xs []processResource) int {
	total := 0.00
	for _, v := range xs {
		total += float64(len(v.Jobs))
	}
	return int(math.Round(total / float64(len(xs))))
}

func jobSort(jobs []job, resources []processResource) []processResource {

	jobsCopy := make([]job, len(jobs))
	copy(jobsCopy, jobs)

	for _, job := range jobsCopy {
		for i, resource := range resources {
			if job.Duration > resource.Capacity {
				continue
			}
			resources[i].Add(job)
			jobs = jobRemove(jobs, job)
			break
		}
	}

	//Resources are overcapacity so allocate best
	if len(jobs) > 0 {
		for len(jobs) > 0 {

			sort.Slice(resources, func(i, j int) bool {
				return resources[i].Capacity > resources[j].Capacity
			})

			for i := range resources {
				resources[i].Add(jobs[0])
				jobs = jobRemove(jobs, jobs[0])
				break
			}
		}
	}

	//Load balance
	if anyEmptyResources(resources) {
		//Some resources have nothing to do so load balance

		sort.Slice(resources, func(i, j int) bool {
			return len(resources[i].Jobs) > len(resources[j].Jobs)
		})

		avgLoad := average(resources)
		idleResources := []processResource{}
		for i, resource := range resources {
			if len(resource.Jobs) == 0 {
				idleResources = append(idleResources, resources[i])
			}
		}

		for _, idleResouce := range idleResources {
			aboveAverageLoad := []processResource{}
			for i, resource := range resources {
				if len(resource.Jobs) > avgLoad {
					aboveAverageLoad = append(aboveAverageLoad, resources[i])
				}
			}

			for _, aboveAverageResource := range aboveAverageLoad {

				work := aboveAverageResource.Jobs[len(aboveAverageResource.Jobs)-1]
				if idleResouce.Capacity-work.Duration > 0 {

					for i, resource := range resources {
						if resource.ID == idleResouce.ID {
							resources[i].Add(work)
						}

						if resource.ID == aboveAverageResource.ID {
							resources[i].Remove(work)
						}

					}
				}

			}
		}

	}

	return resources
}

func process(w http.ResponseWriter, r *http.Request) {
	var body payload
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Body error")
	}

	json.Unmarshal(reqBody, &body)

	var resources []processResource

	maxValue := max(body.Jobs)

	for _, resource := range body.Resources {
		resources = append(resources, processResource{ID: resource.ResourceID, Capacity: maxValue})
	}

	json.NewEncoder(w).Encode(jobSort(body.Jobs, resources))
}

func main() {

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/health", getHealth).Methods("GET")
	router.HandleFunc("/process", process).Methods("POST")
	log.Fatal(http.ListenAndServe(":8080", router))
}
