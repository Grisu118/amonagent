package collectors

import (
	"encoding/json"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/amonapp/amonagent/internal/settings"
	"github.com/amonapp/amonagent/plugins"
)

func (p SystemDataStruct) String() string {
	s, _ := json.Marshal(p)
	return string(s)
}

func (p AllMetricsStruct) String() string {
	s, _ := json.Marshal(p)
	return string(s)

}

func (p HostDataStruct) String() string {
	s, _ := json.Marshal(p)
	return string(s)
}

// AllMetricsStruct -XXX
type AllMetricsStruct struct {
	System    SystemDataStruct `json:"system"`
	Processes ProcessesList    `json:"processes"`
	Host      HostDataStruct   `json:"host"`
	Plugins   interface{}      `json:"plugins"`
	Checks    interface{}      `json:"checks"`
}

// HostDataStruct -XXX
type HostDataStruct struct {
	Host       string       `json:"host"`
	MachineID  string       `json:"machineid"`
	ServerKey  string       `json:"server_key"`
	Distro     DistroStruct `json:"distro"`
	IPAddress  string       `json:"ip_address"`
	InstanceID string       `json:"instance_id"`
}

// SystemDataStruct - collect all system metrics
type SystemDataStruct struct {
	CPU     CPUUsageStruct   `json:"cpu"`
	Network NetworkUsageList `json:"network"`
	Disk    DiskUsageList    `json:"disk"`
	Load    LoadStruct       `json:"loadavg"`
	Uptime  string           `json:"uptime"`
	Memory  MemoryStruct     `json:"memory"`
}

// PluginResultStruct - a channel struct that holds plugin results
type PluginResultStruct struct {
	Name   string
	Result interface{}
}

// CollectPluginsData - XXX
func CollectPluginsData(configuredPlugins []plugins.ConfiguredPlugin) (interface{}, interface{}) {
	PluginResults := make(map[string]interface{})
	var CheckResults interface{}
	var wg sync.WaitGroup

	resultChan := make(chan PluginResultStruct, len(configuredPlugins))

	for _, p := range configuredPlugins {
		wg.Add(1)

		go func(p plugins.ConfiguredPlugin) {
			PluginResult, err := p.Plugin.Collect()
			if err != nil {
				log.Errorf("Can't get stats for plugin: %s", err)
			}

			r := PluginResultStruct{Name: p.Name, Result: PluginResult}

			resultChan <- r
			defer wg.Done()
		}(p)

	}

	wg.Wait()
	close(resultChan)

	for result := range resultChan {
		if result.Name == "checks" {
			CheckResults = result.Result
		} else {
			PluginResults[result.Name] = result.Result
		}

	}

	return PluginResults, CheckResults
}

// CollectHostData - XXX
func CollectHostData() HostDataStruct {

	host := Host()
	// Load settings
	settings := settings.Settings()

	var machineID string
	var InstanceID string
	var ip string
	var distro DistroStruct

	machineID = GetOrCreateMachineID()
	InstanceID = CloudID()
	ip = IPAddress()
	distro = Distro()

	hoststruct := HostDataStruct{
		Host:       host,
		MachineID:  machineID,
		Distro:     distro,
		IPAddress:  ip,
		ServerKey:  settings.ServerKey,
		InstanceID: InstanceID,
	}

	return hoststruct
}

// CollectSystemData - XXX
func CollectSystemData() SystemDataStruct {
	var networkUsage NetworkUsageList
	var cpuUsage CPUUsageStruct
	var diskUsage DiskUsageList
	var memoryUsage MemoryStruct
	var UptimeString string
	var Load LoadStruct

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		networkUsage, _ = NetworkUsage()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		cpuUsage = CPUUsage()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		diskUsage, _ = DiskUsage()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		memoryUsage = MemoryUsage()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		UptimeString = Uptime()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		Load = LoadAverage()
	}()

	wg.Wait()

	SystemData := SystemDataStruct{
		CPU:     cpuUsage,
		Network: networkUsage,
		Disk:    diskUsage,
		Load:    Load,
		Uptime:  UptimeString,
		Memory:  memoryUsage,
	}

	return SystemData

}

// CollectProcessData - XXX
func CollectProcessData() ProcessesList {
	var ProcessesUsage ProcessesList
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ProcessesUsage, _ = Processes()
	}()

	wg.Wait()

	return ProcessesUsage
}

// CollectAllData - XXX
func CollectAllData(configuredPlugins []plugins.ConfiguredPlugin) AllMetricsStruct {

	ProcessesData := CollectProcessData()
	SystemData := CollectSystemData()
	Plugins, Checks := CollectPluginsData(configuredPlugins)
	HostData := CollectHostData()

	allMetrics := AllMetricsStruct{
		System:    SystemData,
		Processes: ProcessesData,
		Host:      HostData,
		Plugins:   Plugins,
		Checks:    Checks,
	}

	return allMetrics
}
