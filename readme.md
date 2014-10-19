#About

Varanus is a very small, very simple system monitoring utility.
It consists of two parts: A client which is running on all systems that should be monitored, and a collector that is running on the system you wish to use to view the information that's being sent by the clients.
The client collects basic information such as CPU load, memory usage, disk usage, and network utilization, and sends it to the configured collector. The collector then displays this information in real-time in your browser.

There's very little setup and no dependencies required!

Varanus is currently officially supported on Linux systems, your mileage may vary with other systems. Other systems may be supported in the future, though!

#Installation

Varanus is written in the [Go programming language](https://golang.org), which makes building it from source a snap.

To build: Make sure you've got a working Go installation first. After that, just issue `go build` in both the `client` and `collector` directories to produce the required binaries!

#FAQs

**Why "Varanus"?**

"Varanus" is the scientific name for the genus of [monitor lizards](http://en.wikipedia.org/wiki/Varanus), and it just sounds cool.

**Why make another monitoring system? There's loads out there already!**

There is loads of monitoring systems and frameworks out there already, yes. However, Varanus is being designed to be both lightweight and incredibly simple to configure and deploy, both for the systems that will be monitored and the system that collects all the monitoring data. There's no heavy list third-party dependencies that need to be set up, and because it's very lightweight, it's suitable for monitoring embedded network-enabled Linux systems, too!