# Backwards Compatibility Policy

Prior to Draft hitting the v1.0 milestone, there are no guarantees for backwards compatibility.

Once Draft hits the v1.0 milestone, Draft will maintain a strong commitment to backward compatibility. All of our changes to protocols and formats are backward compatible from Draft 1.0 until Draft 2.0. No features, flags, or commands are removed or substantially modified (other than bug fixes).

We also try very hard to not change publicly accessible Go library definitions inside of the pkg/ directory of our source code.

For a quick summary of our backward compatibility guidelines for releases between 2.0 and 3.0:

- Command line commands, flags, and arguments MUST be backward compatible
- File formats (such as draft.toml) MUST be backward compatible
- Any application that worked with a previous version of Draft MUST work on a new version of Draft, barring the cases where (a) upstream broke backwards compatibility, and (b) the application worked because it exploited a bug
- Core Maintainers of Go libraries inside of pkg/ SHOULD lean towards remaining backwards compatible. Note that this is NOT a guarantee, but a recommendation. Code inside pkg/ may change if there's a valid reason for changing it.
