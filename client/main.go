package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type config struct {
	ConfigVer     int    // Configuration version
	CollectorAddr string // Address of the collector server, in format addr:port
	LogLevel      int
	Adapter       string
	AuthKey       string
}

type disk struct {
	Total      int    // Total disk space, in kB
	Free       int    // Free disk space, in kB
	Mount      string // Mount location of the volume
	Filesystem string // Filesystem location
}

type stats struct {
	AuthKey   string   // Authentication key
	Hostname  string   // System hostname
	Uptime    int      // System uptime, in seconds
	Processes int      // Process count
	LoadAvg   []string // [One minute, Five minutes, Fifteen minutes, Running tasks/Total tasks]
	MemTotal  int      // Total system memory
	MemFree   int      // Available system memory, not counting buffers or cache
	SwapTotal int      // Total system swap
	SwapFree  int      // Available system swap
	NetDown   int      // Network download speed, in bytes/sec
	NetUp     int      // Network upload speed, in bytes/sec
	Disks     []disk   // Information on all mounted volumes
}

var (
	Config        = new(config)
	Stats         = new(stats)
	ConfigVersion = 1
	Version       = "1"
)

var (
	regexpNumbers   = regexp.MustCompile(`^([0-9]+)$`)
	regexpMemTotal  = regexp.MustCompile(`MemTotal:(?:\s*)([0-9]*)`)
	regexpMemFree   = regexp.MustCompile(`MemFree:(?:\s*)([0-9]*)`)
	regexpBuffers   = regexp.MustCompile(`Buffers:(?:\s*)([0-9]*)`)
	regexpCached    = regexp.MustCompile(`Cached:(?:\s*)([0-9]*)`)
	regexpSwapTotal = regexp.MustCompile(`SwapTotal:(?:\s*)([0-9]*)`)
	regexpSwapFree  = regexp.MustCompile(`SwapFree:(?:\s*)([0-9]*)`)
	regexpNetDev    = regexp.MustCompile(`^(?:.+?): ([0-9]+?)(?:\s+?)(?:[0-9]+?)(?:\s+?)(?:[0-9]+?)(?:\s+?)(?:[0-9]+?)(?:\s+?)(?:[0-9]+?)(?:\s+?)(?:[0-9]+?)(?:\s+?)(?:[0-9]+?)(?:\s+?)(?:[0-9]+?)(?:\s+?)([0-9]+?)(?:\s+?)(?:.+)$`)
	regexpDisk      = regexp.MustCompile(`^(.+?)(?:\s+?)([0-9]+?)(?:\s+?)(?:[0-9]+?)(?:\s+?)([0-9]+?)(?:\s+?)(?:[0-9]+?)%(?:\s+?)(.+)$`)
)

// Pretty-prints a line in the console log, based on the configuration's selected logging level
func writeLog(text string, level int) {
	if Config.LogLevel > level {
		return
	}
	logLevel := []string{"DEBUG", "INFO", "WARNING", "ERROR", "SEVERE"}
	if level >= len(logLevel) {
		level = len(logLevel) - 1
	}
	if level < 0 {
		level = 0
	}
	timestamp := time.Now().UTC().Format("2006/01/02 15:04:05")
	fmt.Println(timestamp+" ["+logLevel[level]+"]", text)
}

// Collects information on: Process count, system uptime, and system load average
func collectCPUInfo() {
	for {
		items, _ := ioutil.ReadDir("/proc")
		processes := 0
		for _, item := range items {
			if !item.IsDir() {
				continue
			}
			if regexpNumbers.MatchString(item.Name()) {
				processes++
			}
		}
		uptimeRaw, _ := ioutil.ReadFile("/proc/uptime")
		uptime, _ := strconv.Atoi(strings.Split(string(uptimeRaw), ".")[0])
		loadAvgRaw, _ := ioutil.ReadFile("/proc/loadavg")
		loadAvg := strings.Split(string(loadAvgRaw), " ")
		Stats.Uptime = uptime
		Stats.LoadAvg = []string{loadAvg[0], loadAvg[1], loadAvg[2], loadAvg[3]}
		Stats.Processes = processes
		time.Sleep(1 * time.Second)
	}
}

// Collects information on: Total memory available, total swap space available, total memory available, and total swap space available
func collectMemInfo() {
	for {
		memInfoRaw, _ := ioutil.ReadFile("/proc/meminfo")
		memInfo := string(memInfoRaw)
		memTotalRaw := regexpMemTotal.FindStringSubmatch(memInfo)
		memFreeRaw := regexpMemFree.FindStringSubmatch(memInfo)
		memBuffersRaw := regexpBuffers.FindStringSubmatch(memInfo)
		memCachedRaw := regexpCached.FindStringSubmatch(memInfo)
		swapTotalRaw := regexpSwapTotal.FindStringSubmatch(memInfo)
		swapFreeRaw := regexpSwapFree.FindStringSubmatch(memInfo)
		memTotal, memFree, swapTotal, swapFree := -1, -1, -1, -1
		memBuffers, memCached := 0, 0
		if memTotalRaw != nil {
			memTotal, _ = strconv.Atoi(memTotalRaw[1])
		}
		if memFreeRaw != nil {
			memFree, _ = strconv.Atoi(memFreeRaw[1])
		}
		if swapTotalRaw != nil {
			swapTotal, _ = strconv.Atoi(swapTotalRaw[1])
		}
		if swapFreeRaw != nil {
			swapFree, _ = strconv.Atoi(swapFreeRaw[1])
		}
		if memBuffersRaw != nil {
			memBuffers, _ = strconv.Atoi(memBuffersRaw[1])
		}
		if memCachedRaw != nil {
			memCached, _ = strconv.Atoi(memCachedRaw[1])
		}
		Stats.MemTotal = memTotal
		Stats.MemFree = memFree + memBuffers + memCached
		Stats.SwapTotal = swapTotal
		Stats.SwapFree = swapFree
		time.Sleep(3 * time.Second)
	}
}

// Collects information on: Network download speed, and network upload speed
func collectNetInfo() {
	for {
		dl, ul := 0, 0
		netRaw, _ := ioutil.ReadFile("/proc/net/dev")
		lines := strings.Split(string(netRaw), "\n")
		for _, line := range lines {
			if !strings.Contains(line, Config.Adapter+": ") {
				continue
			}
			bytes := regexpNetDev.FindStringSubmatch(line)
			if bytes == nil {
				return // TODO: Should be cleaner than just exiting the goroutine
			}
			dl, _ = strconv.Atoi(bytes[1])
			ul, _ = strconv.Atoi(bytes[2])
			break
		}
		time.Sleep(1 * time.Second)
		netRaw, _ = ioutil.ReadFile("/proc/net/dev")
		lines = strings.Split(string(netRaw), "\n")
		for _, line := range lines {
			if !strings.Contains(line, Config.Adapter+": ") {
				continue
			}
			bytes := regexpNetDev.FindStringSubmatch(line)
			if bytes == nil {
				return // TODO: Should be cleaner than just exiting the goroutine
			}
			dlTemp, _ := strconv.Atoi(bytes[1])
			ulTemp, _ := strconv.Atoi(bytes[2])
			Stats.NetDown = dlTemp - dl
			Stats.NetUp = ulTemp - ul
			break
		}
	}
}

// Collects information on: Mounted partitions, and disk usage
func collectDiskInfo() {
	for {
		cmd := exec.Command("df", "--block-size=1000")
		output, err := cmd.Output()
		if err != nil {
			writeLog("Failed to run command: df", 2)
			writeLog(err.(error).Error(), 2)
			continue
		}
		tempDisks := []disk{}
		lines := strings.Split(string(output), "\n")
		for i := 1; i < len(lines) - 1; i++ {
			line := regexpDisk.FindStringSubmatch(lines[i])
			fmt.Println(line)
			total, _ := strconv.Atoi(line[2])
			free, _ := strconv.Atoi(line[3])
			tempDisks = append(tempDisks, disk{Total: total, Free: free, Mount: line[4], Filesystem: line[1]})
		}
		Stats.Disks = tempDisks
		time.Sleep(60 * time.Second)
	}
}

// Collects information on: Authentication key, and system hostname
func collectSystem() {
	Stats.AuthKey = Config.AuthKey
	for {
		Stats.Hostname, _ = os.Hostname()
		time.Sleep(1 * time.Minute)
	}
}

// Encodes and sends the current system stats to the configured collector server
func sendStats() {
	time.Sleep(5 * time.Second) // Let the collection functions gather initial data first
	raddr, _ := net.ResolveUDPAddr("udp", Config.CollectorAddr)
	c, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		writeLog("Unable to open a socket to the configured collector!", 4)
		return
	}
	defer c.Close()
	for {
		packet, _ := json.Marshal(Stats)
		c.Write(packet)
		time.Sleep(1 * time.Second)
	}
}

func main() {
	writeLog("Varanus version "+Version+" is starting", 0)
	configData, err := ioutil.ReadFile("config.json")
	if err != nil {
		writeLog("Failed to read config.json!", 4)
		return
	}
	err = json.Unmarshal(configData, Config)
	if err != nil {
		writeLog("Failed to load config.json! Invalid JSON?", 4)
		return
	}
	go collectCPUInfo()
	go collectMemInfo()
	go collectNetInfo()
	go collectDiskInfo()
	go collectSystem()
	sendStats()
}
