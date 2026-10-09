package resolve

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// model(name) returns the cached class constructed under the configured model
// path. Read the literal name only after the real loader and cache lookup have
// established that contract. A preset's name-only model rule is not needed.
func (r *Resolver) wheelsModelReturn(global string, def *parser.FunctionDef, args [][]parser.Token, baseDir, component string) string {
	name := beanArgument(args, "name", 0)
	if name == "" || strings.ContainsAny(name, "/\\ .") {
		return ""
	}

	if r.wheelsSource(def.URI.Path()).methods["model"].body != wheelsTokens(wheelsModelFactory) {
		return ""
	}

	helper := func(name string) *parser.FunctionDef {
		if component != "" {
			return r.ResolveFunc(component, name, baseDir)
		}

		return r.wheelsPlanFunc(global, name)
	}
	for method, body := range map[string]string{"$createModelClass": wheelsCreateModelClass, "$cachedModelLookup": wheelsCachedModelLookup} {
		fd := helper(method)
		if fd == nil || r.wheelsSource(fd.URI.Path()).methods[strings.ToLower(method)].body != wheelsTokens(body) {
			return ""
		}
	}

	rootFactory := helper("$createObjectFromRoot")
	if rootFactory == nil || !wheelsRootFactoryMatches(r.wheelsSource(rootFactory.URI.Path()).methods["$createobjectfromroot"].body) {
		return ""
	}

	create := helper("$createModelClass")

	plan := r.wheelsSource(create.URI.Path()).producers.get("$createmodelclass")
	if plan == nil || len(plan.parameters) != 3 || !strings.EqualFold(plan.parameters[0], "name") || !strings.EqualFold(plan.parameters[1], "modelPaths") || !strings.EqualFold(plan.parameters[2], "type") || wheelsTokens(plan.defaults["modelpaths"]) != wheelsTokens(`application.wheels.modelPath`) || wheelsTokens(plan.defaults["type"]) != wheelsTokens(`"model"`) {
		return ""
	}

	paths, ok := r.wheelsObjectPaths(baseDir, filepath.Dir(global), "modelPath")
	if !ok {
		return ""
	}

	var targets []string

	for _, config := range paths {
		// The class loader falls back to the final path's base Model.cfc.
		list := strings.Split(config, ",")
		target := ""

		for i, path := range list {
			dir := strings.ReplaceAll(strings.Trim(strings.TrimSpace(path), "/"), "/", ".")
			if dir == "" {
				return ""
			}

			target = r.ComponentPath(dir+"."+name, "")
			if target != "" {
				break
			}

			if i == len(list)-1 {
				target = r.ComponentPath(dir+".Model", "")
			}
		}

		if target == "" {
			return ""
		}

		if !slices.Contains(targets, target) {
			targets = append(targets, target)
		}
	}

	return strings.Join(targets, "|")
}

const wheelsModelFactory = `
local.rv = $cachedModelLookup(name = arguments.name);
if (IsBoolean(local.rv) && !local.rv) {
 return $doubleCheckedLock(condition = "$cachedModelClassExists", conditionArgs = arguments, execute = "$createModelClass", executeArgs = arguments, name = "modelLock#application.applicationName#");
}
return local.rv;`

const wheelsCachedModelLookup = `
if (StructKeyExists(application, "wheels") && StructKeyExists(application.wheels, "models") && StructKeyExists(application.wheels.models, arguments.name)) {
 return application.wheels.models[arguments.name];
}
return false;`

const wheelsCreateModelClass = `
local.modelPathsArray = ListToArray(arguments.modelPaths);
local.iEnd = ArrayLen(local.modelPathsArray);
for (local.i = 1; local.i <= local.iEnd; local.i++) {
 local.modelPath = local.modelPathsArray[local.i];
 local.fileName = $objectFileName(name = arguments.name, objectPath = local.modelPath, type = arguments.type);
 if (local.fileName != arguments.type || local.i == ArrayLen(local.modelPathsArray)) {
  application.wheels.models[arguments.name] = $createObjectFromRoot(path = local.modelPath, fileName = local.fileName, method = "$initModelClass", name = arguments.name);
  local.rv = application.wheels.models[arguments.name];
  break;
 }
}
return local.rv;`
