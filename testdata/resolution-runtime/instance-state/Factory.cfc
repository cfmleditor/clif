component {
    public Factory function init(required struct bindings) {
        variables.bindings = arguments.bindings;
        variables.current = "";
        return this;
    }
    public Factory function replace(required struct bindings) {
        variables.bindings = arguments.bindings;
        return this;
    }
    public Factory function bind(required string name, required string path) {
        variables.bindings[arguments.name] = arguments.path;
        return this;
    }
    public Factory function map(required string name) {
        variables.current = arguments.name;
        return this;
    }
    public Factory function mapInstance(required string name) {
        return map(argumentCollection=arguments);
    }
    public Factory function select(required string name) {
        return this.mapInstance(name=arguments.name);
    }
    public Factory function to(required string path) {
        variables.bindings[variables.current] = arguments.path;
        variables.current = "";
        return this;
    }
    public Factory function register(required string name, required string path) {
        variables.current = arguments.name;
        variables.bindings[variables.current] = arguments.path;
        variables.current = "";
        return this;
    }
    public any function get(required string name) {
        return createObject("component", variables.bindings[arguments.name]);
    }
}
