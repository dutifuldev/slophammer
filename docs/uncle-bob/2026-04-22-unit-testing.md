---
title: "Morning Bathrobe Rant: Unit Testing"
author: Bob <dutifulbob@gmail.com>
date: 2026-04-22
---

# Morning Bathrobe Rant: Unit Testing

| Field             | Value                                                                          |
| ----------------- | ------------------------------------------------------------------------------ |
| Speaker           | Uncle Bob Martin                                                               |
| Source date       | 2026-04-22                                                                     |
| X status          | [2046926727650132292](https://x.com/unclebobmartin/status/2046926727650132292) |
| Transcript method | Whisper `small`, local transcription                                           |

## Transcript

Unit testing.
If you're doing AI you've got to get those AIs to write unit tests.
First thing I put in my agents.md file is test driven development, follow the three laws.
Of course they're like any developer they don't really do it and they kind of half-heartedly try but as soon as you focus them on something they forget.
They're not going to follow those rules properly and they do write some unit tests.
They try, they get some unit tests done but it's not enough.
It's not enough so the next thing I have them do is run coverage analysis.

This is right in their list of procedures so as soon as they get something to work they have to run coverage and then I tell them you must cover everything that's uncovered so they drive the coverage up pretty dog-on-high, not too bad, and then I have them run crap analysis and mutate analysis.
I'll talk about all that stuff later but if you don't have unit tests and you are screwed, you are screwed blue because those AIs will just rip the code that they don't care about apart and they don't care about anything except what they're doing right now.

So you've got to have unit tests so at least they know that they have broken something and you've got to drive the coverage high so that at least they find the things that they have broken.
People who are vibe coding without unit tests, man, there's a comeuppance coming.
