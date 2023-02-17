# HTTPx A Real-World Performance Test and Analysis

HTTP (HyperText Transfer Protocol) is the underlying protocol of the World Wide Web. HTTP has gone through many changes that have helped maintain its simplicity while shaping its flexibility.

<p align="center">
  <img src="https://github.com/SyngoPredevelopment/HTTPx/blob/main/HTTPEvolution.png" />
</p>

<p align="center">Figure 1: Evolution of the HTTP protocol</p>

<br>

HTTP/3 is the third major version of the Hypertext Transfer Protocol used to exchange information on the World Wide Web, complementing the widely-deployed HTTP/1.1 and HTTP/2. Unlike previous versions which relied on the well-established TCP, HTTP/3 uses QUIC, a multiplexed transport protocol built on UDP.

<p align="center">
  <img src="https://github.com/SyngoPredevelopment/HTTPx/blob/main/HTTP3.png" />
</p>

<p align="center">Figure 2: HTTP/2 vs HTTP/3</p>

<br>

QUIC has advantages over TCP in terms of its latency, versatility, and applicationlayer simplicity at the cost of high implementation overhead. However, the difference
in developer design and operator configurations is highly attributed to its performance,
resulting in inconsistencies in performance across implementations and testing results.
A performance test on production endpoints shows that deploying and using QUIC does
not automatically lead to an increase in performance in network and application in many
use cases.
Another study found that QUIC performs well on short flows such as small file download and browsing websites but has reduced throughput over long flows such as large
files download and streaming videos. However, it gives lower latency resulting in a better
video quality video delivery and lower handshake time for web workloads than TLS 1.3
over TCP.

Theory is great, but it’s more convincing if we can see some real-data and real performance improvements of HTTP/3 over HTTP/2 and HTTP/1.1 

This prototype is build exactly with that purpose, it's using only file and network I/O through it's design in order to omit parts which could effect performance heavily (like databases, parsing libraries etc.). The main purpose is to find out the differences of the HTTP versions specially for DICOMweb use cases:

* <b>Retrieve</b> transaction:<br>
Study level: */studies/{study}* <br>
Series level: */studies/{study}/series/{series}* <br>
Instance level: */studies/{study}/series/{series}/instances/{instance}*

* <b>Store</b> transaction: <br>
Study level: */studies/{study}*


## Setup of prototype (Windows, for any other OS it should be similar):
=====

### <b>1. Clone repository locally</b>
`git clone https://github.com/SyngoPredevelopment/HTTPx.git .`

### <b>2. Generate the certificates</b> (in folder cert, only if not available or outdated)
Per default in the repository there are certificates for 127.0.0.1 specially for testing on a local machine, if that's the intention, than this step can be omitted.
Install first mkcert and generate certificates for your server machine (use IP X.X.X.X of server machine).

`choco install mkcert`

`cd ..\cert`

`mkcert localhost 127.0.0.1 X.X.X.X ::1`

Rename certificates to names expected in the prototype:

`move *-key.pem cert-priv.pem`

`move *.pem cert-public.pem`

### <b>3. Build the executables</b> (same for client and server and folder executables)
Install please first golang with:

`choco install golang`

### Build for Windows
`cmd /C "set "CGO_ENABLED=0" && set "GOOS=windows" && set "GOARCH=amd64" && go build -v"`

### Build for Linux
`cmd /C "set "CGO_ENABLED=0" && set "GOOS=linux" && set "GOARCH=amd64" && go build -v"`

### <b>4. Prepare directory</b> structure with DICOM files for server as well as for the client
With the httpx-folder executable an arbitrary folder structure with DICOM files related to a study can be converted to the desired structure needed by the client and server prototype.
The desired structure in folders is the following: `StudyInstanceUid/SeriesInstanceUid/InstanceUid.dcm`

The desired structure can be created with:

`mkdir out`

`httpx-folder -dirin FOLDER-WITH-STUDY-DATA -dirout .\out`

### <b>5. Run the server</b>
The following command with start the DICOMweb server listening on different ports for different HTTP protocol versions:

`httpx-server -v 0 -dir .\out -cert ..`

### Ports used by the server:
`HTTP/1.1:  8080
HTTPS/1.1: 8081
HTTPS/2:   8082
HTTP/3:    8083`

Important parameters for the httpx-server:

`-dir - directory to be used for retrieve (output) or store (input)`

`-cert - directory with public and private certificate: cert-priv.perm, cert-public.pem`

`-v - number for the log level verbosity, 1 - Summary data, 2 - HTTP logs, 3 - debug, 4 - info`

### <b>6. Run the client</b>
Retrieve use case with the different protocol versions:

`httpx-client -v 0 -http 3.0 -operation retrieve -dir .\in https://127.0.0.1:8083/studies/1.3.12.2.1107.5.99.3.30000009052811420737800000003`

`httpx-client -v 0 -http 2.0 -operation retrieve -dir .\in https://127.0.0.1:8082/studies/1.3.12.2.1107.5.99.3.30000009052811420737800000003`

`httpx-client -v 0 -http 1.1 -operation retrieve -dir .\in https://127.0.0.1:8081/studies/1.3.12.2.1107.5.99.3.30000009052811420737800000003`

Send use case with the different protocol versions:

`httpx-client -v 0 -http 3.0 -operation send -chunking single  -mode async -dir d:\in\1.3.12.2.1107.5.99.3.30000012031310075961300000006 https://127.0.0.1:8083/studies/1.3.12.2.1107.5.99.3.30000012031310075961300000006`

`httpx-client -v 0 -http 2.0 -operation send -chunking single  -mode async -dir d:\in\1.3.12.2.1107.5.99.3.30000012031310075961300000006 https://127.0.0.1:8082/studies/1.3.12.2.1107.5.99.3.30000012031310075961300000006`

`httpx-client -v 0 -http 1.1 -operation send -chunking single  -mode async -dir d:\in\1.3.12.2.1107.5.99.3.30000012031310075961300000006 https://127.0.0.1:8081/studies/1.3.12.2.1107.5.99.3.30000012031310075961300000006`

Important parameters for the httpx-client:

`-dir - directory to be used for retrieve (output) or store (input)`

`-chunking - chunking mode to be used: single | multi (default "single" as single part messages)`

`-http - http version to be used: 1.1 | 2.0 | 3.0 (default "1.1")`

`-operation - operation to be executed: retrieve | send (default "retrieve")`

`-mode - mode to be used: sync | async (default "sync"). For async a threadpool with the number of CPUs is used. sync is single threaded.`

`-cert - directory with public and private certificate: cert-priv.perm, cert-public.pem`

`-v - number for the log level verbosity, 2 - HTTP logs, 3 - debug, 4 - info`

<br>
# Disclaimer

THIS PROTOTYPE IS SOLELY FOR PERSONAL, NONCOMMERCIAL USE IN A RESEARCH SETTING AND IS INTENDED FOR DEMONSTRATIONAL PURPOSES ONLY. THE DEMONSTRATIONAL APPLICATION IS STILL IN THE DEVELOPMENT STAGE. THE INFORMATION EXPRESSED IN THIS DEMONSTRATIONAL APPLICATION IS NOT MEDICAL ADVICE. THIS DEMONSTRATIONAL APPLICATION IS NOT A MEDICAL DEVICE AND DOES NOT AND SHOULD NOT BE CONSTRUED TO PROVIDE HEALTH-RELATED OR MEDICAL ADVICE, OR CLINICAL DECISION SUPPORT, OR TO SUPPORT OR REPLACE THE DIAGNOSIS, RECOMMENDATION, ADVICE, TREATMENT, OR DECISION BY AN APPROPRIATELY TRAINED AND LICENSED PHYSICIAN, INCLUDING, WITHOUT LIMITATION, WITH RESPECT TO ANY LIFE SUSTAINING OR LIFESAVING TREATMENT OR DECISION. THIS DEMONSTRATIONAL APPLICATION DOES NOT CREATE A PHYSICIAN-PATIENT RELATIONSHIP BETWEEN JHU AND ANY INDIVIDUAL. BEFORE MAKING ANY MEDICAL OR HEALTH-RELATED DECISIONS, INDIVIDUALS ARE ADVISED TO CONSULT AN APPROPRIATELY TRAINED AND LICENSED PHYSICIAN.
