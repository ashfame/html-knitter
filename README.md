HTML Knitter
============

Takes a HTML file path as input and generates another output HTML file with the following changes:

- Remove all JS code (if specified via `-remove-js` flag)
- Copies over the css files referenced and directly embed them in the HTML source (Doesn't do any optimisation to remove unused CSS)
- Copies over the font files in use and directly embed them in the HTML source and rewrite their references in CSS code.

## Usage

Build it: `make` (You will need the Go toolchain)

Run it: `./html-knitter -input input.html -output output.html -remove-js`

**Note:** Experimental project, not battle-tested in production
