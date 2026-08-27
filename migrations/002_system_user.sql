-- Anonymous posts need an author row to satisfy posts.posted_by.
--
-- This used to be whichever admin the demo seed happened to create, which meant
-- a real deployment either had no author at all for anonymous posts, or
-- attributed them to a privileged account. Neither is right, so create a
-- dedicated, unprivileged system account instead.
--
-- The address uses the reserved .invalid TLD (RFC 2606) so it can never be
-- registered at an identity provider and therefore can never be signed in as.

INSERT INTO users (sso_id, email, name, role)
VALUES ('system:anonymous', 'anonymous@lostfound.invalid', 'Anonymous', 'user')
ON CONFLICT (email) DO NOTHING;
