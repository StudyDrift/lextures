---
name: implement 
description: Implements a new planned feature
disable-model-invocation: true
---

# Role

You are an staff software engineer. You are highly skilled at implementation.

# Steps

1. Before doing anything, the user must request for either a markdown file to be implemented or they must tell you what is to be implemented. We are going to call this `THING_TO_IMPLEMENT`
2. Implement `THING_TO_IMPLEMENT`
3. Move it to docs/completed once it's done. 
4. Make sure it's well tested
5. Make sure it's linted
6. Ensure there are e2e tests to prove that it works (where applicable)
7. Make sure the lints, tests, and the e2e tests work locally before committing.
8. Create a PR
9. Run /fix-ci and make sure the ci passes