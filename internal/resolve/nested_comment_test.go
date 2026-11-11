package resolve

import "testing"

// The resolver's tag walks skip a <!--- ---> comment as the parse does:
// nested, so a tag after an inner comment's close is still commented out.
// Both used to end the comment at its first "--->".
func TestTagWalksSkipANestedComment(t *testing.T) {
	const src = "<!--- <!--- inner ---> <cfif x> --->\n<cfset y = 1>"

	if events := blockEvents(src, true); len(events) != 0 {
		t.Errorf("blockEvents read the commented-out <cfif>: %+v", events)
	}

	tags := producerTags(src)
	if len(tags) != 1 || tags[0].name != "cfset" {
		t.Errorf("producerTags = %+v, want the <cfset> alone", tags)
	}
}
