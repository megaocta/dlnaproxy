# Introduction

This program does exactly what the name suggests: It proxies a DLNA server. Supplied with the IP and port of the HTTP portion of a DLNA media server, it will announce itself on the local network and proxies all connections from the local network to the foreign server.

Perfect for when you want to make your DLNA server accessible for everyone on the same network, without messing with routes and NAT. Makes it possible to connect to a remote DLNA server over a Layer 3 only network, such as Wireguard or TUN OpenVPN.

Tested working with MiniDLNA

Build-all.bat builds for 64-bit Windows, 64-bit Linux, 32-bit ARM and compresses afterwards with UPX.

# Usage 

./dlnaproxy -target=&lt;HTTP ip&gt;:&lt;HTTP port&gt;