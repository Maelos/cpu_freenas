// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !nocpu

package collector

import (
	"io/ioutil"
	"os/exec"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type statCollector struct {
	cpu  typedDesc
	temp typedDesc
}

//get the # of CPU cores and their temperatures and put them in a cpu struct, then return a slice of these structs
func getCPUTemps() (out []float64) {

	// IPMI Variables/Settings
	//useIPMI := true //I want to use IPMI, use what you like
	ipmiHost := "192.168.1.64" // IP address or DNS-resolvable hostname of IPMI server:
	ipmiUser := "root"         // IPMI username
	// IPMI password file. This is a file containing the IPMI user's password
	// on a single line and should have 0600 permissions:
	ipmiPWfromFile, _ := ioutil.ReadFile("/root/ipmi_password") //needs to find the file at location and read the line to the variable
	ipmiPW := string(ipmiPWfromFile)

	//define the command to get the number of CPUs and then use it
	numCPUCmd := exec.Command("/usr/local/bin/ipmitool", " -I lanplus -H ", ipmiHost, " -U ", ipmiUser, " -f ", ipmiPW, " sdr elist all | grep -c -i 'cpu.*temp")
	numCPUSoB, _ := numCPUCmd.Output() //returns a slice of bytes and an error
	numCPU := int(numCPUSoB[0])        //converts the first and hopefully only value of slice of bytes into an int

	//go through each CPU and get the temperature
	if numCPU == 1 {
		//define the command used to get the CPU temperature
		tempCmd := exec.Command("/usr/local/bin/ipmitool", " -I lanplus -H ", ipmiHost, " -U ", ipmiUser, " -f ", ipmiPW, " sdr elist all | grep 'CPU Temp' | awk '{print $10}'")
		temp, _ := tempCmd.Output()
		out = append(out, float64(temp[0]))
	} else {
		for i := 0; i < numCPU; i++ {
			tempCmd := exec.Command("/usr/local/bin/ipmitool", " -I lanplus -H ", ipmiHost, " -U ", ipmiUser, " -f ", ipmiPW, " sdr elist all | grep 'CPU", string(i), " Temp' | awk '{print $10}'")
			temp, _ := tempCmd.Output()
			out = append(out, float64(temp[0]))
		}
	}
	return out // returns the slice of cpuCollector structs
}

func init() {
	registerCollector("cpu", defaultEnabled, NewStatCollector)
}

// NewStatCollector returns a new Collector exposing CPU stats.
func NewStatCollector() (Collector, error) {
	return &statCollector{
		cpu: typedDesc{nodeCPUSecondsDesc, prometheus.CounterValue},
		temp: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "temperature_celsius"),
			"CPU temperature",
			[]string{"cpu"}, nil,
		), prometheus.GaugeValue},
	}, nil
}

// Expose CPU stats using sysctl.
func (c *statCollector) Update(ch chan<- prometheus.Metric) error {

	cpuTemp := getCPUTemps()

	for cpu, t := range cpuTemp {
		lcpu := strconv.Itoa(cpu)
		ch <- c.temp.mustNewConstMetric(temp, lcpu)
	}
	return nil
}