# Checking a blue-green deploy for zero failed requests

Use this to verify the zero-downtime promise of the `blue-green` strategy on a real VM.

1. Deploy a test app with one `deployment` service that has an HTTP health check, for
   example `traefik/whoami`, to an environment with a domain routed to the VM.
2. Start sending traffic for longer than a deploy takes:

   ```sh
   go run ./cmd/loadtest -url http://staging.example.com/ -rate 50 -duration 2m
   ```

3. While it runs, deploy the same build again from the UI (each deploy starts a new
   container, so this exercises the full switch).
4. The tool prints `total=… ok=… non-2xx=… errors=…` and exits non-zero if any request
   failed. A correct blue-green deploy reports `non-2xx=0 errors=0`.

If failures show up, look at the deployment log: the switch should read
"Both versions serve", then "taking … out of routing", "Draining for …" and "Stopping …".
Failures right after "Stopping" mean the app does not finish in-flight requests on
SIGTERM, or the `drain` period is shorter than Traefik needs to reload the route file.
