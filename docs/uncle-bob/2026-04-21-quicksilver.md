---
title: "Morning Bathrobe Rant: Quicksilver"
author: Uncle Bob Martin
date: 2026-04-21
url: https://x.com/unclebobmartin/status/2046578685080220074
transcript_method: "Whisper `small`, local transcription"
---

# Morning Bathrobe Rant: Quicksilver

## Transcript

acceptance tests.
If you ever put your finger on a little puddle of mercury, a little ball of mercury, put a little ball of mercury on the counter, put your finger on it, it'll scoot out from underneath, and then you put your finger on it again, and it'll scoot out in a random direction, and every time you try to put your finger on it, it scoots out in a random direction.
You ever felt like that when you were doing vibacoding?

You tell the AI to give you a feature, and it gives you the feature, then you tell the AI to give you another feature, and it gives you that feature, but it broke the first feature, and then you tell it to fix the first feature, which it does, but then it breaks the second feature, and if you try to add a third feature, it breaks the first two features.
If you ever had that happen to you, it certainly happened to me.
That's when I learned that you have to use acceptance testing.
You must provide a strict formula for the behavior of the system.

You must provide an unbreakable law for the behavior of the system, and you can't just load it into the context.
You can't just say, well, here, read this document and do whatever in the document, because it'll ignore that.
You know, they say AIs, they don't really follow the rules.

They sort of do, but as you will learn, in the context of the AI, the first things you say and the last things you say have a lot more importance than the stuff in the middle, and if you have a long description of what the system is supposed to do, that long description is going to sit in the middle, and it's going to get ignored, so what you need to have are acceptance tests, acceptance tests that are executed outside of the processing of the AI, tools that drive the execution of the acceptance tests.

We've had lots of these over the years, you know, fitness, JBehave, Cucumber, all these tools that allow you to specify the behavior of the system and then verify the behavior of the system through automated tests.
Nowadays, I like to use gherkin, you know, given when then statements.
That's what I like to use.

I like to put them in a file, and then I have the AI build a parser for me, and it parses it down to something like JSON, and then I have the AI build a generator, and the generator takes in the JSON and emits unit tests, and those unit tests all have to pass, and that anchors the behavior of the system.
Every time the AI makes a small change to add a new feature, it has to run all the acceptance tests, and if any of them break it has to fix them all, so that you don't get the little bubble of mercury issue.
