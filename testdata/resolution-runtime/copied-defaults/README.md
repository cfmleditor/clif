# Copied method and default evaluation probe

Serve this directory through the selected CFML test engine and request run.cfm.
Keep the JSON result with the engine name, exact version and test configuration.
No particular result is asserted until supported engine behavior is established.
This fixture was executed on ColdFusion Server 2025,0,10,331899. The normalized
observations are saved in observations-cf2025.json. Both copied defaults and
bodies used Host when called through the Host slot. Both extracted local
references failed to find the default helper. The Provider controls used Provider;
supplied arguments stayed explicit while the copied body used Host.

Execution used the existing local image tassjenkins_coldfusion_compile:latest,
image ID sha256:47cad7a08017d960a62369558e33e720df5b5edd9a4a840fe28b87bb505a761f.
The task container had network disabled and this directory mounted read-only
at /app. Its entrypoint was overridden with /bin/sh; it started
/opt/ColdFusion/cfusion/bin/coldfusion and requested /run.cfm locally with curl.
The container was removed after the run. No image was pulled. An initial mount
at the default wwwroot did not run the fixture because this image's configured
document root is /app; that failed task container was also removed.

Provider and Host return distinct helper values. The copied methods match the
two source forms still outstanding in Wheels: an interpolated string default
and a direct helper-call default. Each method also reads a helper in its body,
so a default's context can be compared with the body's context. Supplying the
argument checks the path where its default is not evaluated. Extracted local
references are observed separately from invocation through the receiving slot.

These observations alone do not establish startup ownership of application.wo,
plugin replacement rules, engine-adapter state or Mura's registry snapshot.
They close only the method-binding/default-evaluation question for the recorded
engine and version. Other engines and deployed Wheels ownership remain unproved.
The resolver continues to withhold copied default contexts.
