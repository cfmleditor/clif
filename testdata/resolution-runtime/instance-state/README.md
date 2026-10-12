# Private instance state probe

Executed on ColdFusion Server 2025,0,10,331899 using the existing local image
tassjenkins_coldfusion_compile:latest, image ID
sha256:47cad7a08017d960a62369558e33e720df5b5edd9a4a840fe28b87bb505a761f.
The disposable container had network disabled and this directory mounted
read-only at /app. The /bin/sh entrypoint started
/opt/ColdFusion/cfusion/bin/coldfusion; curl requested
http://127.0.0.1:8500/run.cfm inside the container. Docker --rm removed it.
No image was pulled. Normalized observations are in observations-cf2025.json.
The probe was repeated with init and the fluent setters declaring Factory as
their return type. All normalized results remained identical. The Go regression
also consumes this exact typed Factory source.
The probe was extended again with mapInstance forwarding argumentCollection=arguments
to map, and select forwarding a named argument to this.mapInstance. Chaining select
to to changed the binding to B; chaining mapInstance to to changed it back to A.
Both forwarding methods declare Factory and execute the delegated state changes.

Independent binding records produced A and B. Replacement changed the first
factory's lookup to B, and reversing the replacement changed it back to A.
Registering a literal ID in an empty table produced A; rebinding the same ID
with different letter case produced B.
The extended probe also executed map(name).to(path): a fresh binding produced A,
case-insensitive rebinding produced B, and a setter that changed its pending key
then wrote the table in the same method produced A. Two factories retained
separate pending keys across interleaved map/to calls; the first returned A.
Mutating the caller's original binding struct also changed the factory's later
lookup from A to B. That last observation supports withholding a literal-record
snapshot when the record is passed by reference rather than supplied inline.
It does not certify other engines, arbitrary binder APIs or framework startup.
