---
name: workflow
description: >
  Perform any development work on the ambient-ui component. Full
  workflow instructions.
---

# Ambient UI Development workflow

Strict workflow execution to be used when performing any ambient-ui component work.

## User Input

```text
$ARGUMENTS
```

## Workflow

### 1. Align with Specs

All work must be aligned with specs. If the user or instructions are asking
for work to be done without a verified matching section in a spec, STOP. 

Before proceeding with implementation, update the relevant spec to match
the request.

### 2. Perform Test Driven Development

Determine the size of the work. If it is large, break it down into smaller atomic units of work and execute
this workflow for each one. You can use subagents to handle this gracefully, preserving your context.

Define tests that are relevant and that, when they pass, provide a genuine trust
that the system is properly implemented in a way that aligns with the spec and the user's intent.

Then, proceed with methodical implementation, breaking work into atomic units of work for subagents
when possible. Bias towards using subagents for implementation tasks, but ensure they read this workflow skill to understand
the development standards.

Test and verify implementation before proceeding to the next step.

#### Standards

These standards are to be used during all ambient-ui component development work.

##### Fakes over Mocks

When testing, fakes are to be preferred over mocks. Tests that don't test reality are not tests at all.

##### Accessibility

- For Red Hat assets, small text (17pt or smaller) needs to have a 4.5:1 contrast ratio at minimum. Large text (18pt or larger) and informative graphics like icons need to have a 3:1 contrast ratio at minimum.
- Saturated hues that are similar in intensity can vibrate when used together, creating fuzziness around the edges like a glowing blur. These colors can be difficult for anyone to look at and distinguish, and even painful for those with vision differences.
  An example of high-contrast colors that do not vibrate.

  Do this: Combine bright colors with less saturated and neutral colors.
  Image showing misuse: An example of low-contrast colors that vibrate.

  Not this: Don’t use colors with similar intensity in the same area.


##### Technology

- React/Next.js
- shadcn/ui components
- Tabbed views MUST persist the active tab in the URL via `?tab=` query parameter using controlled `<Tabs value={activeTab}>` with useState + useEffect + replaceState. Ensures tabs survive reload and support deep linking.

##### Colors

Core Palette

| Color Name | HEX Code | RGB | CMYK | Pantone / Notes |
| :--- | :--- | :--- | :--- | :--- |
| **red-05** | `#fef0f0` | — | — | — |
| **red-10** | `#fce3e3` | — | — | — |
| **red-20** | `#fbc5c5` | — | — | — |
| **red-30** | `#f9a8a8` | — | — | — |
| **red-40** | `#f56e6e` | — | — | — |
| **red-50** | `#ee0000` | 238 0 0 | 0 98 85 0 | Pantone 1788C (Red Hat red) |
| **red-60** | `#a60000` | — | — | — |
| **red-70** | `#5f0000` | — | — | — |
| **red-80** | `#3f0000` | — | — | — |
| **white** | `#ffffff` | 255 255 255 | 0 0 0 0 | Pantone White |
| **gray-10** | `#f2f2f2` | — | — | — |
| **gray-20** | `#e0e0e0` | — | — | — |
| **gray-30** | `#c7c7c7` | — | — | — |
| **gray-40** | `#a3a3a3` | — | — | — |
| **gray-45** | `#8c8c8c` | — | — | — |
| **gray-50** | `#707070` | — | — | — |
| **gray-60** | `#4d4d4d` | — | — | — |
| **gray-70** | `#383838` | — | — | — |
| **gray-80** | `#292929` | — | — | — |
| **gray-90** | `#1f1f1f` | — | — | — |
| **gray-95** | `#151515` | — | — | ux black |
| **black** | `#000000` | 0 0 0 | 60 40 40 100 | Pantone Black C |

---

Secondary Palette

| Color Name | HEX Code | RGB | CMYK | Pantone / Notes |
| :--- | :--- | :--- | :--- | :--- |
| **orange-10** | `#ffe8cc` | — | — | — |
| **orange-20** | `#fccb8f` | — | — | — |
| **orange-30** | `#f8ae54` | — | — | — |
| **orange-40** | `#f5921b` | 245 146 27 | 0 50 100 0 | Pantone 144C |
| **orange-50** | `#ca6c0f` | — | — | — |
| **orange-60** | `#9e4a06` | — | — | — |
| **orange-70** | `#732e00` | — | — | — |
| **orange-80** | `#4d1f00` | — | — | — |
| **yellow-10** | `#fff4cc` | — | — | — |
| **yellow-20** | `#ffe072` | — | — | — |
| **yellow-30** | `#ffcc17` | 248 204 23 | 0 15 100 0 | Pantone 108C |
| **yellow-40** | `#dca614` | — | — | — |
| **yellow-50** | `#b98412` | — | — | — |
| **yellow-60** | `#96640f` | — | — | — |
| **yellow-70** | `#73480b` | — | — | — |
| **yellow-80** | `#54330b` | — | — | — |
| **teal-10** | `#daf2f2` | — | — | — |
| **teal-20** | `#b9e5e5` | — | — | — |
| **teal-30** | `#9ad8d8` | — | — | — |
| **teal-40** | `#63bdbd` | — | — | — |
| **teal-50** | `#37a3a3` | 55 163 163 | 80 10 30 10 | Pantone 2234C |
| **teal-60** | `#147878` | — | — | — |
| **teal-70** | `#004d4d` | — | — | — |
| **teal-80** | `#003333` | — | — | — |
| **purple-10** | `#ece6ff` | — | — | — |
| **purple-20** | `#d0c5f4` | — | — | — |
| **purple-30** | `#b6a6e9` | — | — | — |
| **purple-40** | `#876fd4` | — | — | — |
| **purple-50** | `#5e40be` | 94 64 190 | 85 80 0 0 | Pantone 2097C |
| **purple-60** | `#3d2785` | — | — | — |
| **purple-70** | `#21134d` | — | — | — |
| **purple-80** | `#1b0d33` | — | — | — |

> *Note: The "Auxiliary colors" section in your text was empty, so it has been omitted here.*

---

Information Palette
> ⚠️ **Usage Note:** These colors are utilitarian. They should only be used for their intended purposes, not for decorative visuals.

| Color Name | HEX Code | RGB | CMYK | Pantone / Notes |
| :--- | :--- | :--- | :--- | :--- |
| **success-green-10** | `#e9f7df` | — | — | — |
| **success-green-20** | `#d1f1bb` | — | — | — |
| **success-green-30** | `#afdc8f` | — | — | — |
| **success-green-40** | `#87bb62` | — | — | — |
| **success-green-50** | `#63993d` | 99 153 61 | 70 0 100 10 | Pantone 7737C |
| **success-green-60** | `#3d7317` | — | — | — |
| **success-green-70** | `#204d00` | — | — | — |
| **success-green-80** | `#183301` | — | — | — |
| **danger-orange-10** | `#ffe3d9` | — | — | — |
| **danger-orange-20** | `#fbbea8` | — | — | — |
| **danger-orange-30** | `#f89b78` | — | — | — |
| **danger-orange-40** | `#f4784a` | — | — | — |
| **danger-orange-50** | `#f0561d` | 240 86 29 | 0 83 100 0 | Pantone 165C |
| **danger-orange-60** | `#b1380b` | — | — | — |
| **danger-orange-70** | `#731f00` | — | — | — |
| **danger-orange-80** | `#4c1405` | — | — | — |
| **interaction-blue-10**| `#e0f0ff` | — | — | — |
| **interaction-blue-20**| `#b9dafc` | — | — | — |
| **interaction-blue-30**| `#92c5f9` | — | — | — |
| **interaction-blue-40**| `#4394e5` | — | — | — |
| **interaction-blue-50**| `#0066cc` | 0 102 204 | 85 55 0 5 | Pantone 2387C |
| **interaction-blue-60**| `#004d99` | — | — | — |
| **interaction-blue-70**| `#003366` | — | — | — |
| **interaction-blue-80**| `#032142` | — | — | — |

**Color	Association	More info**
red 	Red Hat	Red is our brand color. Do not use red to represent negative things.
danger-orange 	Error, decrease, or failure	Something negative has occurred, like a destructive error or a decrease in value.
orange 	Caution	A non-destructive action or error has occurred.
yellow 	Warning	Take action now to avoid a destructive action or error.
success-green 	Success, increase	Something positive has occurred, like a successful action or increase in value.
teal 	General or neutral	A button or information has no severity.
interaction-blue 	Link or interaction	Clicking the object or text leads to a hyperlink or state change.
purple 	Info or note tip	Helpful information is available.
gray 	Null	A button or information is unavailable or unimportant.


##### Typefaces

**Red Hat Text** takes all of the personality from Display and optimizes it for more demanding applications. As the name implies, it’s easier to read in paragraphs or when text is used at small sizes—like in a whitepaper or tooltip.

To increase readability, Text has more height difference between the upper- and lowercase letters, more space between narrow characters, and more variation in letterform stroke weights.

**Red Hat Mono**

Red Hat Mono was created to distinguish code from natural-language text. Mono stands for monospaced, meaning each letter takes up the same amount of horizontal space. This creates neat columns of text and makes scanning code easier.

Mono should only be used when demonstrating code snippets in communications and interfaces, or as a stylistic approach for a more technical audience like the Red Hat Developer Program or the Code Comments podcast.


##### General Principles 

- Use consistent design tokens
- Use iconography to improve the user experience and ability to scan the page
- Everything must be accessible
- Use reusable components
- Unless specified by the user, NEVER MAKE CONSIDERATIONS BASED ON A NEED FOR BACKWARDS COMPATIBILITY.
- Use atomic, conventional commits.

### 3. Perform Critical Review

Launch the following agents in parallel to review and critique your work:

1. Security review agent: 
- Expert in OWASP vulnerability classes
- Thoroughly vets for supply chain vulnerabilities

2. Spec alignment review agent: 
- Thoroughly review the relevant spec & the implementation. Flags any gaps, no matter how trivial. Spec is the source of truth. 

3. Architecture review agent: 
- Identifies architectural inconsistencies, deviations from standards (domain driven design, domain oriented observability, using ports/adapters, etc.)
- Identifies duplicate code, magic strings/numbers, etc.

4. Code quality agent:
- Identify lint & formatting issues
- Identify dead code

5. Documentation sync agent: 
- Ensure relevant README & documentation reflects the changes in this unit of work
- Do not be overly verbose. Follow any project documentation guidelines, aim for conciseness. Do not over document. 

6. UX review agent:
- Embodies the persona of Steve Krug and the book "Don't Make Me Think"
- Identifies _anything_ that might make the user think, is poorly organized, doesn't follow principles, etc.
- Uses playwright to see and interact with the ui to verify & discover all findings. 

7. User target demographic review agent:
- Embodies the persona of an engineer that is using the platform to:
    - Observe units of work, their status, and the overall status of the system (is human intervention necessary etc.)
    - Define and maintain credentials, binding access to projects and specific agents. 
- Exercises the application from end to end, following real user patterns to discover system breaking bugs.

### 4. Address Review Comments

Use subagent(s) to address relevant review comments. Although you are to use your judgement about 
whether a review comment is relevant, you are also to be picky about your work. Take pride in doing
the correct thing, even if it's not the easiest.

After addressing review comments, loop back to step 3.

If step 3 provides no actionable comments, proceed to the next step. Do not worry if you spend
many cycles in this stage. Correctness is worth the effort.

### 5. Request User Feedback

Get the UI in a running state so that the user can provide feedback. There are two
ways for you to do this:

#### Running inside ambient

If you are running within the ambient platform, you will have instructions
for building images, and deploying to an OpenShift cluster. You should do this,
create a route, and add the route to the relevant annotation (see your operating instructions.)

#### Running locally

If you are running locally, simply follow the local development setup, and provide
the user with the URL to open in their browser to provide feedback. Let them know
what to test and where to look.