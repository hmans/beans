# Claude Code extras

When working with Claude Code, the team here at Beans HQ uses some sensible rules (see below)
together [its CC's Agent Skills system](https://code.claude.com/docs/en/skills) to make Beans fly.

## Option 1: All the Beans, all the time

If you don't use another ticketing system and want to go all in on Beans, then put this rule in the
project (or global) `CLAUDE.md`:

```markdown
## Issue Tracking for Agents with Beans
 
**IMPORTANT**: If you received a Beans primer during startup, then this project uses **Beans** for
agentic issue tracking, and you MUST use your `issue-tracking-with-beans` skill.
```

("Beans primer" means the output of the `beans prompt` command you put
[in the `SessionStart` hook](/README.md#agent-configuration).)

The first time Claude sees any mention of "beans", it should read
[`issue-tracking-with-beans` skill.](/extras/claude/skills/issue-tracking-with-beans/SKILL.md). See
the skill file for more.

## Option 2: Linear (or GitHub Issues or whatever) + Beans

Some of us use both an external ticketing system, and those weirdos use an alternate rule. Here's an
example rule (and linked skill) for Linear, which should be easy to adapt for your own ticketing
system:

```markdown
## Issue Tracking: Linear for humans, Beans for Agents

**IMPORTANT**: If you received a Beans primer during startup, then this project uses both **Linear**
and **Beans** for agentic issue tracking, and you MUST use your
`issue-tracking-with-beans-and-linear` skill.
```

Here, the
[`issue-tracking-with-beans-and-linear` skill](/extras/claude/skills/issue-tracking-with-beans-and-linear/SKILL.md)
is called.

**PLEASE NOTE:** These two skills are mutually exclusive. It's okay to put them both in your
`~/.claude/skills/` folder but **only one of the two rules** should be used in each project.
