# Introduction

In this tutorial, we will create a sample project that will be automatically validated, built, and published by Catalyst Forge.
During this process, we will encounter the various tools and configurations that power Catalyst Forge.
By the end of this tutorial, you will be equipped to begin developing your own projects.

We will be building a trivial program in the Python language in order to facilitate learning.
Python was chosen due to its simplicity and more well-known nature.
Understanding the Python language is not required, and more adventerous learners may choose to substitute the code with their
language of choice.
Catalyst Forge is language agnostic; all that is required is knowing how to build and validate your language of choice.

## Pre-requisites

!!! note

    External contributors will only be able to partially complete this tutorial.
    This is due to the fact that permissions on most repositories (including the playground) do not allow external contributors to
    arbitrarily merge code.
    If you're an external contributor, feel free to follow the tutorial up to the point where merging is required.

Prior to starting this tutorial, please ensure you have the following available on your machine:

1. The latest version of the [Forge CLI](https://github.com/input-output-hk/catalyst-forge/releases)
2. A recent version of [Earthly](https://earthly.dev/) installed and configured

While it's possible to follow along with this tutorial without knowledge of Earthly, it's recommended readers go through the
[Learn the Basics](https://docs.earthly.dev/basics) tutorial in the Earthly documentation prior to proceeding.
A good portion of the tutorial involves building out an `Earthfile`, and having an understanding of the syntax will improve the
learning process.