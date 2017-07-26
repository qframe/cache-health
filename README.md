# cache-health
Cache to keep health information around and serves it at HTTP endpoint

## Health Endpoint

This creates a health endpoint on `:8123`.

```
$ go run main.go
2017/07/26 13:42:22.340385 [  INFO] Dispatch broadcast for Back, Data and Tick
2017/07/26 13:42:22.341392 [NOTICE]          health Name:health     >> Start plugin v0.0.0
2017/07/26 13:42:22.341482 [  INFO]          health Name:health     >> Start ticker 'health-ticker' with duration of 2500ms
2017/07/26 13:42:22.342211 [  INFO]          health Name:health     >> Start health-endpoint: 0.0.0.0:8123
2017/07/26 13:42:22.588183 [II] Start listener for: 'gracious_perlman' [ee191cd6faed7f4719a68b607d9c5771ad5aafc690c6a63b91f99a648d260c35]
2017/07/26 13:42:24.343265 [  INFO]          health Name:health     >> Received HealthBeat: {{0.1.2  2017-07-26 08:59:38 +0000 UTC 0 [stats] true map[]} statsRoutine ee191cd6faed7f4719a68b607d9c5771ad5aafc690c6a63b91f99a648d260c35 start}
2017/07/26 13:42:24.343329 [  INFO]          health Name:health     >> Received HealthBeat: {{0.1.2  2017-07-26 08:59:38 +0000 UTC 0 [logs] true map[]} logSkipRoutine ee191cd6faed7f4719a68b607d9c5771ad5aafc690c6a63b91f99a648d260c35 start}
```
The endpoint looks like this, as the logs of the logging container are skipped, it shows up under skipped - otherwise a log-loop would be created.
```
$ curl -s http://localhost:8123/health |jq '"health:"+.health+" | healthMsg: "+.healthMsg'
"health:healthy | healthMsg: RunningContainers:1 | metricsGoRoutines:1 | logsGoRoutine:(0 [logs] + 1 [skipped])"
$ curl -s http://localhost:8123/health |jq '"stats:"+.statsRoutines+" | log: "+.logRoutines +" | logSkip: "+.logSkipRoutines'
"stats:ee191cd6faed7f4719a68b607d9c5771ad5aafc690c6a63b91f99a648d260c35 | log:  | logSkip: ee191cd6faed7f4719a68b607d9c5771ad5aafc690c6a63b91f99a648d260c35"
```
When starting a new container...
```
$ docker run -e SKIP_ENTRYPOINTS=true -ti --rm --no-healthcheck qnib/uplain-init sleep 15
[II] qnib/init-plain script v0.4.28
> execute CMD 'sleep 15'
```
... the endpoint changes...
```
$ curl -s http://localhost:8123/health |jq '"stats:"+.statsRoutines+" | log: "+.logRoutines +" | logSkip: "+.logSkipRoutines'
"stats:54859ad125ea7b18286b8592882b090807bcd35efffdae3f147e6baba7912624,ee191cd6faed7f4719a68b607d9c5771ad5aafc690c6a63b91f99a648d260c35 | log: 54859ad125ea7b18286b8592882b090807bcd35efffdae3f147e6baba7912624 | logSkip: ee191cd6faed7f4719a68b607d9c5771ad5aafc690c6a63b91f99a648d260c35"
```

In case the container counts are not matching, the health becomes `unhealthy`.