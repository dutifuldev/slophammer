---
title: "Morning Bathrobe Rant: Mutation Testing"
author: Uncle Bob Martin
date: 2026-05-05
url: https://x.com/unclebobmartin/status/2051615123605733752
transcript_method: "Whisper `small`, local transcription"
---

# Morning Bathrobe Rant: Mutation Testing

## Transcript

Mutation testing.
So let's say that you've convinced your agents to write tests.
You know you put something in an MD file that said you will write tests and of course the agent says yes I will and then then you're you're having it implement features and things and sure enough it's writing tests.
You think it's doing a good job with those tests do you?
You think it always follows that rule?
You think it doesn't forget sometimes?
They do.
You know, the farther that the instruction gets into the context window the less it's going to follow it.

So okay fine you you get your feature done and it looks like it's working and maybe you're a little concerned that it's not as tested as it ought to be and so you fire up a coverage analysis tool which is always a good idea and you get you get the coverage tool and it says no you got about 80% line coverage there and you you tell the agent to beef up the line coverage and it does it writes more tests goes through and it says oh yeah I didn't test this bit and I didn't test that bit and then you get the coverage up to like 98% and you think you know that's pretty good I think it's okay but you're forgetting something.

Coverage does not mean something was tested.
Coverage only means that something was executed.
There could be all kinds of missing assertions.
That's what a mutation tester is for.
A mutation tester scans the source code looking for things it can change.
It will change less thans, into greater thans, equals into not equals, assignments into nulls.
It'll say oh this line says x equals f of y and it'll change it to x equals null or x equals zero.

It'll do all kinds of nasty little changes inside the code and then it will run the tests for each one of those nasty little changes.
It will run the tests and if the tests pass well that is a surviving mutant and it must be killed.
The mutation tester will report the surviving mutants and then you can tell the agent okay now kill those mutants don't let them survive and it will write more tests.

It will write tests that plug those holes and in the end you will wind up with a test suite that tests everything and if you have a good suite of acceptance tests now you know that the system behaves properly and now you know that the unit tests test virtually everything.
It's a very good way to have excellent code coverage but more than code coverage you get semantic stability.
