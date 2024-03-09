# ⚙️ Compressing CS2 Demos
> My blog post: [Compressing CS2 Demos](https://healeycodes.com/compressing-cs2-demos)

<br>

A practical example of how to compress a CS2 demo.

Download the `pera-vs-system5-m1-vertigo.dem` demo from https://www.hltv.org/matches/2370182/pera-vs-system5-esl-challenger-league-season-47-europe and store it at the root.

Run `go run .` and check the size of the produced files (`naive.json`, `better.json`, and `better.proto`).

To generate protobuf code, run `protoc --go_out=. optimal.proto`.
