package main

var docstring = `
== Description
GETs a given URL and sends result (if success) or error (if not) to the corresponding ports

"success" is any response with HTTP status code 200
"error" is any other response or protocol error (e.g., timeout)

== Ports
=== IN
Expects URL as a string

=== RES
On success, [url, body] multipart is sent over this port

=== ERR
On error, [url, status, body] multipart is sent over this port
if protocol error status = 1, body = error string

== Meta arguments
`
