# fw-p — unresolved findings and candidates

Every finding below is still reported. A candidate is a liberal match offered for a person to weigh, never an answer the resolver took: **high** is the only component declaring every method the function calls on the receiver (or the only one, named like it); **medium** is a sole match on weaker evidence, or the one named like the receiver among several; **low** is one of several. *Defined* is how many indexed files declare a method of that name: 0 means it is missing from the workspace, more means the resolver could not connect the call to it.

| Category | Findings | high | medium | low | none | method defined nowhere |
|---|---:|---:|---:|---:|---:|---:|
| variable | 269 | 80 | 26 | 127 | 36 | 27 |
| return-type | 65 | 0 | 27 | 38 | 0 | 0 |
| method | 20 | 0 | 9 | 3 | 8 | 8 |
| object | 1 | 0 | 0 | 0 | 1 | 0 |

## Variable definitions — a receiver whose component is unknown

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 18 | `user → examples/userManagerAccessControl/model/beans/user.cfc` | 14 | low | declares getId(), getFirstName(), getLastName(); named like the receiver 'user' (3 candidates) | `examples/userManagerAccessControl/controllers/login.cfc:25` variable 'user' has no component ref |
| 15 | `rc.question → examples/qBall/model/beans/question.cfc` | 14 | medium | declares getId(); named like the receiver 'question' (14 candidates) | `examples/qBall/controllers/question.cfc:65` variable 'rc.question' has no component ref |
| 12 | `q → examples/qBall/model/beans/question.cfc` | 14 | high | declares getId(), getTitle(), getAnswered(), getUser(), getCreated() | `examples/qBall/views/main/default.cfm:23` variable 'q' has no component ref |
| 11 | `answer → examples/qBall/model/beans/answer.cfc` | 1 | high | declares getSelectedAnswer(), getUser(), getCreated(), getId(), getDisapprovers(), getApprovers(), getText(); named like the receiver 'answer' | `examples/qBall/views/question/view.cfm:22` variable 'answer' has no component ref |
| 11 | `local.user → examples/userManagerAccessControl/model/beans/user.cfc` | 14 | high | declares getId(), getFirstName(), getLastName(), getEmail(), getDepartmentId(), getRoleId(); named like the receiver 'user' | `examples/userManagerAccessControl/views/user/form.cfm:10` variable 'local.user' has no component ref |
| 11 | `variables.question → examples/qBall/controllers/question.cfc` | 14 | low | declares list(); named like the receiver 'question' (14 candidates) | `examples/qBall/controllers/main.cfc:13` variable 'variables.question' has no component ref |
| 11 | `variables.targetbean` | 0 | none |  | `framework/beanProxy.cfc:788` variable 'variables.targetBean' has no component ref |
| 10 | `variables.userservice → examples/userManagerAccessControl/model/services/user.cfc` | 1 | high | declares getByEmail(), validatePassword() | `examples/userManagerAccessControl/controllers/login.cfc:23` variable 'variables.userService' has no component ref |
| 9 | `local.user → examples/userManager/model/beans/user.cfc` | 14 | low | declares getId(), getFirstName(), getLastName(), getEmail(), getDepartmentId(); named like the receiver 'user' (3 candidates) | `examples/userManager/views/user/form.cfm:9` variable 'local.user' has no component ref |
| 9 | `local.user → examples/userManagerAJAX/model/beans/user.cfc` | 14 | low | declares getId(), getFirstName(), getLastName(), getEmail(), getDepartmentId(); named like the receiver 'user' (3 candidates) | `examples/userManagerAJAX/views/user/form.cfm:9` variable 'local.user' has no component ref |
| 9 | `resp` | 0 | none |  | `framework/one.cfc:883` variable 'resp' has no component ref |
| 8 | `arguments.interceptor.bean → framework/beanProxy.cfc` | 1 | medium | declares _inject() | `framework/beanProxy.cfc:487` variable 'arguments.interceptor.bean' has no component ref |
| 6 | `variables.parent → framework/WireBoxAdapter.cfc` | 2 | low | declares containsBean() (2 candidates) | `framework/ioc.cfc:91` variable 'variables.parent' has no component ref |
| 5 | `user → examples/userManager/model/beans/user.cfc` | 3 | low | declares setDepartmentId(), setDepartment(); named like the receiver 'user' (3 candidates) | `examples/userManager/controllers/user.cfc:37` variable 'user' has no component ref |
| 5 | `user → examples/userManagerAJAX/model/beans/user.cfc` | 3 | low | declares setDepartmentId(), setDepartment(); named like the receiver 'user' (3 candidates) | `examples/userManagerAJAX/controllers/user.cfc:35` variable 'user' has no component ref |
| 5 | `variables.departmentservice → examples/userManagerAccessControl/model/services/department.cfc` | 11 | low | declares get() (11 candidates) | `examples/userManagerAccessControl/controllers/user.cfc:47` variable 'variables.departmentService' has no component ref |
| 5 | `variables.userservice → examples/userManager/controllers/user.cfc` | 8 | low | declares delete() (8 candidates) | `examples/userManager/controllers/user.cfc:15` variable 'variables.userService' has no component ref |
| 4 | `data.viadi1 → tests/extrabeans/sheep/default.cfc` | 1 | high | declares getSimple(), getTyped(), getDefaulted(), getDefaultedType() | `tests/defaultPropertyTest.cfc:19` variable 'data.viaDI1' has no component ref |
| 4 | `data.vianew → tests/extrabeans/sheep/default.cfc` | 1 | high | declares getSimple(), getTyped(), getDefaulted(), getDefaultedType() | `tests/defaultPropertyTest.cfc:15` variable 'data.viaNew' has no component ref |
| 4 | `rc.user → examples/userManagerAccessControl/model/beans/user.cfc` | 3 | high | declares setDepartmentId(), setDepartment(), setPasswordHash(), setPasswordSalt(); named like the receiver 'user' | `examples/userManagerAccessControl/controllers/user.cfc:46` variable 'rc.user' has no component ref |
| 4 | `todoservice → examples/todos/controllers/main.cfc` | 11 | low | declares get() (11 candidates) | `examples/todos/controllers/main.cfc:14` variable 'todoservice' has no component ref |
| 4 | `variables.parent` | 1 | none |  | `framework/ioc.cfc:700` variable 'variables.parent' has no component ref |
| 4 | `variables.targetbean → framework/beanProxy.cfc` | 1 | high | declares $methodExists(), $call() | `framework/beanProxy.cfc:83` variable 'variables.targetBean' has no component ref |
| 3 | `arguments.entity → examples/qBall/model/beans/answer.cfc` | 3 | low | declares setCreated(), setEdited() (3 candidates) | `examples/qBall/model/beans/eventHandler.cfc:10` variable 'arguments.entity' has no component ref |
| 3 | `session.auth.user → examples/userManagerAccessControl/model/beans/user.cfc` | 14 | low | declares getId(); named like the receiver 'user' (14 candidates) | `examples/userManagerAccessControl/controllers/main.cfc:10` variable 'session.auth.user' has no component ref |
| 3 | `trace.msg` | 0 | none |  | `framework/one.cfc:1774` variable 'trace.msg' has no component ref |
| 3 | `variables.departmentservice → examples/userManager/model/services/department.cfc` | 11 | low | declares get() (11 candidates) | `examples/userManager/model/services/user.cfc:23` variable 'variables.departmentService' has no component ref |
| 3 | `variables.userservice → examples/userManagerAJAX/controllers/user.cfc` | 8 | low | declares delete() (8 candidates) | `examples/userManagerAJAX/controllers/user.cfc:16` variable 'variables.userService' has no component ref |
| 2 | `arguments.targetbean → framework/beanProxy.cfc` | 1 | medium | declares $call() | `framework/beanProxy.cfc:243` variable 'arguments.targetBean' has no component ref |
| 2 | `department → examples/userManagerAccessControl/model/beans/department.cfc` | 14 | low | declares getId(); named like the receiver 'department' (14 candidates) | `examples/userManagerAccessControl/model/beans/user.cfc:22` variable 'department' has no component ref |
| 2 | `head.listener` | 2 | none |  | `framework/ioc.cfc:684` variable 'head.listener' has no component ref |
| 2 | `local.dept → examples/userManager/model/beans/department.cfc` | 6 | low | declares getName() (6 candidates) | `examples/userManager/views/user/form.cfm:35` variable 'local.dept' has no component ref |
| 2 | `local.dept → examples/userManagerAJAX/model/beans/department.cfc` | 6 | low | declares getName() (6 candidates) | `examples/userManagerAJAX/views/user/form.cfm:35` variable 'local.dept' has no component ref |
| 2 | `local.dept → examples/userManagerAccessControl/model/beans/department.cfc` | 6 | low | declares getName() (6 candidates) | `examples/userManagerAccessControl/views/user/form.cfm:36` variable 'local.dept' has no component ref |
| 2 | `local.role → examples/userManagerAccessControl/model/beans/role.cfc` | 6 | medium | declares getName(); named like the receiver 'role' (6 candidates) | `examples/userManagerAccessControl/views/user/form.cfm:53` variable 'local.role' has no component ref |
| 2 | `q → examples/qBall/model/beans/user.cfc` | 5 | medium | declares getUsername(); named like the receiver 'User' (5 candidates) | `examples/qBall/views/main/default.cfm:28` variable 'q' has no component ref |
| 2 | `rc.question → examples/qBall/model/beans/user.cfc` | 5 | high | declares getUsername(), getId(); named like the receiver 'User' | `examples/qBall/views/question/view.cfm:7` variable 'rc.question' has no component ref |
| 2 | `role → examples/userManagerAccessControl/model/beans/role.cfc` | 14 | medium | declares getId(); named like the receiver 'role' (14 candidates) | `examples/userManagerAccessControl/model/beans/user.cfc:28` variable 'role' has no component ref |
| 2 | `sys.out` | 1 | none |  | `framework/ioc.cfc:654` variable 'sys.out' has no component ref |
| 2 | `this.logservice → tests/aop/services/Log.cfc` | 2 | low | declares logMessage() (2 candidates) | `tests/aop/interceptors/example/Logger.cfc:8` variable 'this.logService' has no component ref |
| 2 | `this.logservice → tests/issue518/Log.cfc` | 2 | low | declares logMessage() (2 candidates) | `tests/issue518/interceptors/example/Logger.cfc:8` variable 'this.logService' has no component ref |
| 2 | `variables.departmentservice → examples/userManager/controllers/user.cfc` | 14 | low | declares list() (14 candidates) | `examples/userManager/controllers/user.cfc:22` variable 'variables.departmentService' has no component ref |
| 2 | `variables.mainservice → examples/subsystems/errortest/controllers/main.cfc` | 13 | low | declares default() (13 candidates) | `examples/subsystems/errortest/controllers/main.cfc:15` variable 'variables.mainService' has no component ref |
| 2 | `variables.user → examples/qBall/controllers/user.cfc` | 2 | low | declares authenticate(); named like the receiver 'user' (2 candidates) | `examples/qBall/controllers/user.cfc:29` variable 'variables.user' has no component ref |
| 2 | `variables.userservice → examples/userManagerAccessControl/controllers/user.cfc` | 14 | low | declares list() (14 candidates) | `examples/userManagerAccessControl/controllers/user.cfc:28` variable 'variables.userService' has no component ref |
| 1 | `adduser → examples/qBall/model/beans/answer.cfc` | 14 | low | declares getId() (14 candidates) | `examples/qBall/controllers/user.cfc:79` variable 'addUser' has no component ref |
| 1 | `answer → examples/qBall/model/beans/user.cfc` | 5 | medium | declares getUsername(); named like the receiver 'User' (5 candidates) | `examples/qBall/views/question/view.cfm:30` variable 'answer' has no component ref |
| 1 | `arguments.interceptor.bean` | 0 | none |  | `framework/beanProxy.cfc:476` variable 'arguments.interceptor.bean' has no component ref |
| 1 | `arguments.target → tests/aop/interceptors/aop/interceptor.cfc` | 3 | low | declares getStackLogService() (3 candidates) | `tests/aop/interceptors/aop/AroundInterceptor.cfc:22` variable 'arguments.target' has no component ref |
| 1 | `arguments.target → tests/issue518/interceptors/aop/interceptor.cfc` | 3 | low | declares getStackLog() (3 candidates) | `tests/issue518/interceptors/aop/AroundInterceptor.cfc:22` variable 'arguments.target' has no component ref |
| 1 | `cfc → examples/modular/handlers/main.cfc` | 2 | low | declares setFramework() (2 candidates) | `framework/one.cfc:1949` variable 'cfc' has no component ref |
| 1 | `departmentservice → examples/userManagerAccessControl/model/services/department.cfc` | 11 | low | declares get() (11 candidates) | `examples/userManagerAccessControl/Application.cfc:36` variable 'departmentService' has no component ref |
| 1 | `head` | 0 | none |  | `framework/ioc.cfc:682` variable 'head' has no component ref |
| 1 | `inteceptor.bean → examples/skinning/controllers/skin.cfc` | 8 | low | declares before() (8 candidates) | `framework/beanProxy.cfc:922` variable 'inteceptor.bean' has no component ref |
| 1 | `interceptor.bean → framework/Application.cfc` | 8 | low | declares onError() (8 candidates) | `framework/beanProxy.cfc:937` variable 'interceptor.bean' has no component ref |
| 1 | `local.interceptor.bean → tests/aop/interceptors/aop/AfterInterceptor.cfc` | 4 | low | declares after() (4 candidates) | `framework/beanProxy.cfc:877` variable 'local.interceptor.bean' has no component ref |
| 1 | `local.user → examples/userManager/model/beans/department.cfc` | 6 | low | declares getName(); named like the receiver 'Department' (6 candidates) | `examples/userManager/views/user/list.cfm:27` variable 'local.user' has no component ref |
| 1 | `local.user → examples/userManagerAJAX/model/beans/department.cfc` | 6 | low | declares getName(); named like the receiver 'Department' (6 candidates) | `examples/userManagerAJAX/views/user/list.cfm:27` variable 'local.user' has no component ref |
| 1 | `local.user → examples/userManagerAccessControl/model/beans/department.cfc` | 6 | low | declares getName(); named like the receiver 'Department' (6 candidates) | `examples/userManagerAccessControl/views/user/list.cfm:27` variable 'local.user' has no component ref |
| 1 | `out` | 0 | none |  | `framework/one.cfc:953` variable 'out' has no component ref |
| 1 | `previnterceptor.bean → framework/beanProxy.cfc` | 1 | medium | declares _inject() | `framework/beanProxy.cfc:500` variable 'prevInterceptor.bean' has no component ref |
| 1 | `question → examples/qBall/model/beans/question.cfc` | 1 | high | declares addAnswer(); named like the receiver 'question' | `examples/qBall/model/services/question.cfc:45` variable 'question' has no component ref |
| 1 | `rc.data → examples/qBall/model/beans/answer.cfc` | 14 | low | declares getId() (14 candidates) | `examples/qBall/controllers/question.cfc:50` variable 'rc.data' has no component ref |
| 1 | `rc.user → examples/qBall/model/beans/user.cfc` | 14 | low | declares getId(); named like the receiver 'user' (14 candidates) | `examples/qBall/controllers/user.cfc:37` variable 'rc.user' has no component ref |
| 1 | `variables.aroundinterceptors[].bean` | 0 | none |  | `framework/beanProxy.cfc:901` variable 'variables.aroundInterceptors[].bean' has no component ref |
| 1 | `variables.beanproxy → framework/beanProxy.cfc` | 1 | high | declares runStacks(); named like the receiver 'beanProxy' | `framework/beanProxy.cfc:333` variable 'variables.beanProxy' has no component ref |
| 1 | `variables.departmentservice → examples/userManagerAJAX/controllers/user.cfc` | 14 | low | declares list() (14 candidates) | `examples/userManagerAJAX/controllers/user.cfc:23` variable 'variables.departmentService' has no component ref |
| 1 | `variables.departmentservice → examples/userManagerAJAX/model/services/department.cfc` | 11 | low | declares get() (11 candidates) | `examples/userManagerAJAX/controllers/user.cfc:36` variable 'variables.departmentService' has no component ref |
| 1 | `variables.departmentservice → examples/userManagerAccessControl/controllers/user.cfc` | 14 | low | declares list() (14 candidates) | `examples/userManagerAccessControl/controllers/user.cfc:23` variable 'variables.departmentService' has no component ref |
| 1 | `variables.formatterservice → skeleton/model/services/formatter.cfc` | 1 | medium | declares longdate() | `skeleton/controllers/main.cfc:13` variable 'variables.formatterService' has no component ref |
| 1 | `variables.friendlyservice → examples/wirebox/model/friendlyservice.cfc` | 1 | high | declares greeting(); named like the receiver 'friendlyservice' | `examples/wirebox/controllers/main.cfc:6` variable 'variables.friendlyservice' has no component ref |
| 1 | `variables.greetingservice → examples/subsystems/5helloservice/model/services/greeting.cfc` | 1 | medium | declares greet() | `examples/subsystems/5helloservice/controllers/main.cfc:5` variable 'variables.greetingService' has no component ref |
| 1 | `variables.nextinterceptor` | 0 | none |  | `framework/beanProxy.cfc:247` variable 'variables.nextInterceptor' has no component ref |
| 1 | `variables.parent → framework/ioc.cfc` | 1 | medium | declares isSingleton() | `framework/ioc.cfc:284` variable 'variables.parent' has no component ref |
| 1 | `variables.roleservice → examples/userManagerAccessControl/controllers/user.cfc` | 14 | low | declares list() (14 candidates) | `examples/userManagerAccessControl/controllers/user.cfc:24` variable 'variables.roleService' has no component ref |
| 1 | `variables.roleservice → examples/userManagerAccessControl/model/services/department.cfc` | 11 | low | declares get() (11 candidates) | `examples/userManagerAccessControl/model/services/user.cfc:107` variable 'variables.roleService' has no component ref |
| 1 | `variables.user → examples/qBall/model/services/user.cfc` | 11 | low | declares get(); named like the receiver 'user' (11 candidates) | `examples/qBall/controllers/user.cfc:12` variable 'variables.user' has no component ref |
| 1 | `variables.userservice → examples/userManagerAJAX/model/services/department.cfc` | 11 | low | declares get() (11 candidates) | `examples/userManagerAJAX/controllers/user.cfc:22` variable 'variables.userService' has no component ref |
| 1 | `variables.userservice → examples/userManagerAJAX/model/services/user.cfc` | 11 | low | declares get(), save() (6 candidates) | `examples/userManagerAJAX/controllers/user.cfc:32` variable 'variables.userService' has no component ref |
| 1 | `variables.userservice → examples/userManagerAccessControl/model/services/department.cfc` | 11 | low | declares get() (11 candidates) | `examples/userManagerAccessControl/controllers/user.cfc:21` variable 'variables.userService' has no component ref |
| 1 | `ws → examples/remote/remote/Test.cfc` | 2 | medium | declares test() | `examples/remote/views/main/default.cfm:6` variable 'ws' has no component ref |

<details><summary>Groups with several candidates</summary>

- `user → examples/userManagerAccessControl/model/beans/user.cfc` — 18 finding(s), 3 candidate(s):
  - low `examples/userManagerAccessControl/model/beans/user.cfc` — declares getId(), getFirstName(), getLastName(); named like the receiver 'user'
  - low `examples/userManager/model/beans/user.cfc` — declares getId(), getFirstName(), getLastName(); named like the receiver 'user'
  - low `examples/userManagerAJAX/model/beans/user.cfc` — declares getId(), getFirstName(), getLastName(); named like the receiver 'user'
- `rc.question → examples/qBall/model/beans/question.cfc` — 15 finding(s), 14 candidate(s):
  - medium `examples/qBall/model/beans/question.cfc` — declares getId(); named like the receiver 'question'
  - low `examples/qBall/model/beans/answer.cfc` — declares getId()
  - low `examples/qBall/model/beans/user.cfc` — declares getId()
  - low `examples/userManager/model/beans/department.cfc` — declares getId()
  - low `examples/userManager/model/beans/user.cfc` — declares getId()
  - low `examples/userManagerAJAX/model/beans/department.cfc` — declares getId()
  - low `examples/userManagerAJAX/model/beans/user.cfc` — declares getId()
  - low `examples/userManagerAccessControl/model/beans/department.cfc` — declares getId()
  - … 6 more
- `variables.question → examples/qBall/controllers/question.cfc` — 11 finding(s), 14 candidate(s):
  - low `examples/qBall/controllers/question.cfc` — declares list(); named like the receiver 'question'
  - low `examples/qBall/model/services/question.cfc` — declares list(); named like the receiver 'question'
  - low `examples/todos/controllers/main.cfc` — declares list()
  - low `examples/userManager/controllers/user.cfc` — declares list()
  - low `examples/userManagerAJAX/controllers/user.cfc` — declares list()
  - low `examples/userManagerAccessControl/controllers/user.cfc` — declares list()
  - low `examples/todos/model/services/todo.cfc` — declares list()
  - low `examples/userManager/model/services/department.cfc` — declares list()
  - … 6 more
- `local.user → examples/userManager/model/beans/user.cfc` — 9 finding(s), 3 candidate(s):
  - low `examples/userManager/model/beans/user.cfc` — declares getId(), getFirstName(), getLastName(), getEmail(), getDepartmentId(); named like the receiver 'user'
  - low `examples/userManagerAJAX/model/beans/user.cfc` — declares getId(), getFirstName(), getLastName(), getEmail(), getDepartmentId(); named like the receiver 'user'
  - low `examples/userManagerAccessControl/model/beans/user.cfc` — declares getId(), getFirstName(), getLastName(), getEmail(), getDepartmentId(); named like the receiver 'user'
- `local.user → examples/userManagerAJAX/model/beans/user.cfc` — 9 finding(s), 3 candidate(s):
  - low `examples/userManagerAJAX/model/beans/user.cfc` — declares getId(), getFirstName(), getLastName(), getEmail(), getDepartmentId(); named like the receiver 'user'
  - low `examples/userManager/model/beans/user.cfc` — declares getId(), getFirstName(), getLastName(), getEmail(), getDepartmentId(); named like the receiver 'user'
  - low `examples/userManagerAccessControl/model/beans/user.cfc` — declares getId(), getFirstName(), getLastName(), getEmail(), getDepartmentId(); named like the receiver 'user'
- `variables.parent → framework/WireBoxAdapter.cfc` — 6 finding(s), 2 candidate(s):
  - low `framework/WireBoxAdapter.cfc` — declares containsBean()
  - low `framework/ioc.cfc` — declares containsBean()
- `user → examples/userManager/model/beans/user.cfc` — 5 finding(s), 3 candidate(s):
  - low `examples/userManager/model/beans/user.cfc` — declares setDepartmentId(), setDepartment(); named like the receiver 'user'
  - low `examples/userManagerAJAX/model/beans/user.cfc` — declares setDepartmentId(), setDepartment(); named like the receiver 'user'
  - low `examples/userManagerAccessControl/model/beans/user.cfc` — declares setDepartmentId(), setDepartment(); named like the receiver 'user'
- `user → examples/userManagerAJAX/model/beans/user.cfc` — 5 finding(s), 3 candidate(s):
  - low `examples/userManagerAJAX/model/beans/user.cfc` — declares setDepartmentId(), setDepartment(); named like the receiver 'user'
  - low `examples/userManager/model/beans/user.cfc` — declares setDepartmentId(), setDepartment(); named like the receiver 'user'
  - low `examples/userManagerAccessControl/model/beans/user.cfc` — declares setDepartmentId(), setDepartment(); named like the receiver 'user'
- `variables.departmentservice → examples/userManagerAccessControl/model/services/department.cfc` — 5 finding(s), 11 candidate(s):
  - low `examples/userManagerAccessControl/model/services/department.cfc` — declares get()
  - low `examples/userManagerAccessControl/model/services/role.cfc` — declares get()
  - low `examples/userManagerAccessControl/model/services/user.cfc` — declares get()
  - low `examples/todos/controllers/main.cfc` — declares get()
  - low `examples/userManager/controllers/user.cfc` — declares get()
  - low `examples/qBall/model/services/user.cfc` — declares get()
  - low `examples/todos/model/services/todo.cfc` — declares get()
  - low `examples/userManager/model/services/department.cfc` — declares get()
  - … 3 more
- `variables.userservice → examples/userManager/controllers/user.cfc` — 5 finding(s), 8 candidate(s):
  - low `examples/userManager/controllers/user.cfc` — declares delete()
  - low `examples/userManager/model/services/user.cfc` — declares delete()
  - low `examples/todos/controllers/main.cfc` — declares delete()
  - low `examples/userManagerAJAX/controllers/user.cfc` — declares delete()
  - low `examples/userManagerAccessControl/controllers/user.cfc` — declares delete()
  - low `examples/todos/model/services/todo.cfc` — declares delete()
  - low `examples/userManagerAJAX/model/services/user.cfc` — declares delete()
  - low `examples/userManagerAccessControl/model/services/user.cfc` — declares delete()
- `todoservice → examples/todos/controllers/main.cfc` — 4 finding(s), 11 candidate(s):
  - low `examples/todos/controllers/main.cfc` — declares get()
  - low `examples/todos/model/services/todo.cfc` — declares get()
  - low `examples/userManager/controllers/user.cfc` — declares get()
  - low `examples/qBall/model/services/user.cfc` — declares get()
  - low `examples/userManager/model/services/department.cfc` — declares get()
  - low `examples/userManager/model/services/user.cfc` — declares get()
  - low `examples/userManagerAJAX/model/services/department.cfc` — declares get()
  - low `examples/userManagerAJAX/model/services/user.cfc` — declares get()
  - … 3 more
- `arguments.entity → examples/qBall/model/beans/answer.cfc` — 3 finding(s), 3 candidate(s):
  - low `examples/qBall/model/beans/answer.cfc` — declares setCreated(), setEdited()
  - low `examples/qBall/model/beans/question.cfc` — declares setCreated(), setEdited()
  - low `examples/qBall/model/beans/user.cfc` — declares setCreated(), setEdited()
- `session.auth.user → examples/userManagerAccessControl/model/beans/user.cfc` — 3 finding(s), 14 candidate(s):
  - low `examples/userManagerAccessControl/model/beans/user.cfc` — declares getId(); named like the receiver 'user'
  - low `examples/qBall/model/beans/user.cfc` — declares getId(); named like the receiver 'user'
  - low `examples/userManager/model/beans/user.cfc` — declares getId(); named like the receiver 'user'
  - low `examples/userManagerAJAX/model/beans/user.cfc` — declares getId(); named like the receiver 'user'
  - low `tests/services/user.cfc` — declares getId(); named like the receiver 'user'
  - low `examples/userManagerAccessControl/model/beans/department.cfc` — declares getId()
  - low `examples/userManagerAccessControl/model/beans/role.cfc` — declares getId()
  - low `examples/qBall/model/beans/answer.cfc` — declares getId()
  - … 6 more
- `variables.departmentservice → examples/userManager/model/services/department.cfc` — 3 finding(s), 11 candidate(s):
  - low `examples/userManager/model/services/department.cfc` — declares get()
  - low `examples/userManager/model/services/user.cfc` — declares get()
  - low `examples/userManager/controllers/user.cfc` — declares get()
  - low `examples/todos/controllers/main.cfc` — declares get()
  - low `examples/qBall/model/services/user.cfc` — declares get()
  - low `examples/todos/model/services/todo.cfc` — declares get()
  - low `examples/userManagerAJAX/model/services/department.cfc` — declares get()
  - low `examples/userManagerAJAX/model/services/user.cfc` — declares get()
  - … 3 more
- `variables.userservice → examples/userManagerAJAX/controllers/user.cfc` — 3 finding(s), 8 candidate(s):
  - low `examples/userManagerAJAX/controllers/user.cfc` — declares delete()
  - low `examples/userManagerAJAX/model/services/user.cfc` — declares delete()
  - low `examples/todos/controllers/main.cfc` — declares delete()
  - low `examples/userManager/controllers/user.cfc` — declares delete()
  - low `examples/userManagerAccessControl/controllers/user.cfc` — declares delete()
  - low `examples/todos/model/services/todo.cfc` — declares delete()
  - low `examples/userManager/model/services/user.cfc` — declares delete()
  - low `examples/userManagerAccessControl/model/services/user.cfc` — declares delete()
- `department → examples/userManagerAccessControl/model/beans/department.cfc` — 2 finding(s), 14 candidate(s):
  - low `examples/userManagerAccessControl/model/beans/department.cfc` — declares getId(); named like the receiver 'department'
  - low `examples/userManager/model/beans/department.cfc` — declares getId(); named like the receiver 'department'
  - low `examples/userManagerAJAX/model/beans/department.cfc` — declares getId(); named like the receiver 'department'
  - low `examples/userManagerAccessControl/model/beans/role.cfc` — declares getId()
  - low `examples/userManagerAccessControl/model/beans/user.cfc` — declares getId()
  - low `examples/qBall/model/beans/answer.cfc` — declares getId()
  - low `examples/qBall/model/beans/question.cfc` — declares getId()
  - low `examples/qBall/model/beans/user.cfc` — declares getId()
  - … 6 more
- `local.dept → examples/userManager/model/beans/department.cfc` — 2 finding(s), 6 candidate(s):
  - low `examples/userManager/model/beans/department.cfc` — declares getName()
  - low `examples/userManagerAJAX/model/beans/department.cfc` — declares getName()
  - low `examples/userManagerAccessControl/model/beans/department.cfc` — declares getName()
  - low `examples/userManagerAccessControl/model/beans/role.cfc` — declares getName()
  - low `tests/declared/things/myconfig.cfc` — declares getName()
  - low `tests/model/beans/user37c.cfc` — declares getName()
- `local.dept → examples/userManagerAJAX/model/beans/department.cfc` — 2 finding(s), 6 candidate(s):
  - low `examples/userManagerAJAX/model/beans/department.cfc` — declares getName()
  - low `examples/userManager/model/beans/department.cfc` — declares getName()
  - low `examples/userManagerAccessControl/model/beans/department.cfc` — declares getName()
  - low `examples/userManagerAccessControl/model/beans/role.cfc` — declares getName()
  - low `tests/declared/things/myconfig.cfc` — declares getName()
  - low `tests/model/beans/user37c.cfc` — declares getName()
- `local.dept → examples/userManagerAccessControl/model/beans/department.cfc` — 2 finding(s), 6 candidate(s):
  - low `examples/userManagerAccessControl/model/beans/department.cfc` — declares getName()
  - low `examples/userManagerAccessControl/model/beans/role.cfc` — declares getName()
  - low `examples/userManager/model/beans/department.cfc` — declares getName()
  - low `examples/userManagerAJAX/model/beans/department.cfc` — declares getName()
  - low `tests/declared/things/myconfig.cfc` — declares getName()
  - low `tests/model/beans/user37c.cfc` — declares getName()
- `local.role → examples/userManagerAccessControl/model/beans/role.cfc` — 2 finding(s), 6 candidate(s):
  - medium `examples/userManagerAccessControl/model/beans/role.cfc` — declares getName(); named like the receiver 'role'
  - low `examples/userManagerAccessControl/model/beans/department.cfc` — declares getName()
  - low `examples/userManager/model/beans/department.cfc` — declares getName()
  - low `examples/userManagerAJAX/model/beans/department.cfc` — declares getName()
  - low `tests/declared/things/myconfig.cfc` — declares getName()
  - low `tests/model/beans/user37c.cfc` — declares getName()
- `q → examples/qBall/model/beans/user.cfc` — 2 finding(s), 5 candidate(s):
  - medium `examples/qBall/model/beans/user.cfc` — declares getUsername(); named like the receiver 'User'
  - low `tests/stubs/UserOneLevel.cfc` — declares getUsername()
  - low `tests/stubs/UserThreeLevel.cfc` — declares getUsername()
  - low `tests/stubs/UserTwoLevel.cfc` — declares getUsername()
  - low `tests/model/beans/user2Bean.cfc` — declares getUsername()
- `role → examples/userManagerAccessControl/model/beans/role.cfc` — 2 finding(s), 14 candidate(s):
  - medium `examples/userManagerAccessControl/model/beans/role.cfc` — declares getId(); named like the receiver 'role'
  - low `examples/userManagerAccessControl/model/beans/department.cfc` — declares getId()
  - low `examples/userManagerAccessControl/model/beans/user.cfc` — declares getId()
  - low `examples/qBall/model/beans/answer.cfc` — declares getId()
  - low `examples/qBall/model/beans/question.cfc` — declares getId()
  - low `examples/qBall/model/beans/user.cfc` — declares getId()
  - low `examples/userManager/model/beans/department.cfc` — declares getId()
  - low `examples/userManager/model/beans/user.cfc` — declares getId()
  - … 6 more
- `this.logservice → tests/aop/services/Log.cfc` — 2 finding(s), 2 candidate(s):
  - low `tests/aop/services/Log.cfc` — declares logMessage()
  - low `tests/issue518/Log.cfc` — declares logMessage()
- `this.logservice → tests/issue518/Log.cfc` — 2 finding(s), 2 candidate(s):
  - low `tests/issue518/Log.cfc` — declares logMessage()
  - low `tests/aop/services/Log.cfc` — declares logMessage()
- `variables.departmentservice → examples/userManager/controllers/user.cfc` — 2 finding(s), 14 candidate(s):
  - low `examples/userManager/controllers/user.cfc` — declares list()
  - low `examples/userManager/model/services/department.cfc` — declares list()
  - low `examples/userManager/model/services/user.cfc` — declares list()
  - low `examples/qBall/controllers/question.cfc` — declares list()
  - low `examples/todos/controllers/main.cfc` — declares list()
  - low `examples/userManagerAJAX/controllers/user.cfc` — declares list()
  - low `examples/userManagerAccessControl/controllers/user.cfc` — declares list()
  - low `examples/qBall/model/services/question.cfc` — declares list()
  - … 6 more
- `variables.mainservice → examples/subsystems/errortest/controllers/main.cfc` — 2 finding(s), 13 candidate(s):
  - low `examples/subsystems/errortest/controllers/main.cfc` — declares default()
  - low `examples/subsystems/errortest/services/main.cfc` — declares default()
  - low `examples/subsystems/4hellocontroller/controllers/main.cfc` — declares default()
  - low `examples/subsystems/5helloservice/controllers/main.cfc` — declares default()
  - low `examples/modular/handlers/main.cfc` — declares default()
  - low `examples/mustache/controllers/main.cfc` — declares default()
  - low `examples/qBall/controllers/main.cfc` — declares default()
  - low `examples/todos/controllers/main.cfc` — declares default()
  - … 5 more
- `variables.user → examples/qBall/controllers/user.cfc` — 2 finding(s), 2 candidate(s):
  - low `examples/qBall/controllers/user.cfc` — declares authenticate(); named like the receiver 'user'
  - low `examples/qBall/model/services/user.cfc` — declares authenticate(); named like the receiver 'user'
- `variables.userservice → examples/userManagerAccessControl/controllers/user.cfc` — 2 finding(s), 14 candidate(s):
  - low `examples/userManagerAccessControl/controllers/user.cfc` — declares list()
  - low `examples/userManagerAccessControl/model/services/department.cfc` — declares list()
  - low `examples/userManagerAccessControl/model/services/role.cfc` — declares list()
  - low `examples/userManagerAccessControl/model/services/user.cfc` — declares list()
  - low `examples/qBall/controllers/question.cfc` — declares list()
  - low `examples/todos/controllers/main.cfc` — declares list()
  - low `examples/userManager/controllers/user.cfc` — declares list()
  - low `examples/userManagerAJAX/controllers/user.cfc` — declares list()
  - … 6 more
- `adduser → examples/qBall/model/beans/answer.cfc` — 1 finding(s), 14 candidate(s):
  - low `examples/qBall/model/beans/answer.cfc` — declares getId()
  - low `examples/qBall/model/beans/question.cfc` — declares getId()
  - low `examples/qBall/model/beans/user.cfc` — declares getId()
  - low `examples/userManager/model/beans/department.cfc` — declares getId()
  - low `examples/userManager/model/beans/user.cfc` — declares getId()
  - low `examples/userManagerAJAX/model/beans/department.cfc` — declares getId()
  - low `examples/userManagerAJAX/model/beans/user.cfc` — declares getId()
  - low `examples/userManagerAccessControl/model/beans/department.cfc` — declares getId()
  - … 6 more
- `answer → examples/qBall/model/beans/user.cfc` — 1 finding(s), 5 candidate(s):
  - medium `examples/qBall/model/beans/user.cfc` — declares getUsername(); named like the receiver 'User'
  - low `tests/stubs/UserOneLevel.cfc` — declares getUsername()
  - low `tests/stubs/UserThreeLevel.cfc` — declares getUsername()
  - low `tests/stubs/UserTwoLevel.cfc` — declares getUsername()
  - low `tests/model/beans/user2Bean.cfc` — declares getUsername()
- `arguments.target → tests/aop/interceptors/aop/interceptor.cfc` — 1 finding(s), 3 candidate(s):
  - low `tests/aop/interceptors/aop/interceptor.cfc` — declares getStackLogService()
  - low `tests/aop/services/service.cfc` — declares getStackLogService()
  - low `tests/aop/services/stringService.cfc` — declares getStackLogService()
- `arguments.target → tests/issue518/interceptors/aop/interceptor.cfc` — 1 finding(s), 3 candidate(s):
  - low `tests/issue518/interceptors/aop/interceptor.cfc` — declares getStackLog()
  - low `tests/issue518/service.cfc` — declares getStackLog()
  - low `tests/issue518/string.cfc` — declares getStackLog()
- `cfc → examples/modular/handlers/main.cfc` — 1 finding(s), 2 candidate(s):
  - low `examples/modular/handlers/main.cfc` — declares setFramework()
  - low `examples/todos/controllers/main.cfc` — declares setFramework()
- `departmentservice → examples/userManagerAccessControl/model/services/department.cfc` — 1 finding(s), 11 candidate(s):
  - low `examples/userManagerAccessControl/model/services/department.cfc` — declares get()
  - low `examples/userManagerAccessControl/model/services/role.cfc` — declares get()
  - low `examples/userManagerAccessControl/model/services/user.cfc` — declares get()
  - low `examples/todos/controllers/main.cfc` — declares get()
  - low `examples/userManager/controllers/user.cfc` — declares get()
  - low `examples/qBall/model/services/user.cfc` — declares get()
  - low `examples/todos/model/services/todo.cfc` — declares get()
  - low `examples/userManager/model/services/department.cfc` — declares get()
  - … 3 more
- `inteceptor.bean → examples/skinning/controllers/skin.cfc` — 1 finding(s), 8 candidate(s):
  - low `examples/skinning/controllers/skin.cfc` — declares before()
  - low `examples/userManagerAccessControl/controllers/login.cfc` — declares before()
  - low `examples/userManagerAccessControl/controllers/user.cfc` — declares before()
  - low `examples/subsystems/errortest/controllers/main.cfc` — declares before()
  - low `tests/aop/interceptors/aop/BeforeInterceptor.cfc` — declares before()
  - low `tests/aop/interceptors/example/Logger.cfc` — declares before()
  - low `tests/issue518/interceptors/aop/BeforeInterceptor.cfc` — declares before()
  - low `tests/issue518/interceptors/example/Logger.cfc` — declares before()
- `interceptor.bean → framework/Application.cfc` — 1 finding(s), 8 candidate(s):
  - low `framework/Application.cfc` — declares onError()
  - low `framework/one.cfc` — declares onError()
  - low `Application.cfc` — declares onError()
  - low `examples/Application.cfc` — declares onError()
  - low `examples/rest/Application.cfc` — declares onError()
  - low `examples/userManager/Application.cfc` — declares onError()
  - low `tests/aop/interceptors/aop/ErrorInterceptor.cfc` — declares onError()
  - low `tests/issue518/interceptors/aop/ErrorInterceptor.cfc` — declares onError()
- `local.interceptor.bean → tests/aop/interceptors/aop/AfterInterceptor.cfc` — 1 finding(s), 4 candidate(s):
  - low `tests/aop/interceptors/aop/AfterInterceptor.cfc` — declares after()
  - low `tests/aop/interceptors/example/Logger.cfc` — declares after()
  - low `tests/issue518/interceptors/aop/AfterInterceptor.cfc` — declares after()
  - low `tests/issue518/interceptors/example/Logger.cfc` — declares after()
- `local.user → examples/userManager/model/beans/department.cfc` — 1 finding(s), 6 candidate(s):
  - low `examples/userManager/model/beans/department.cfc` — declares getName(); named like the receiver 'Department'
  - low `examples/userManagerAJAX/model/beans/department.cfc` — declares getName(); named like the receiver 'Department'
  - low `examples/userManagerAccessControl/model/beans/department.cfc` — declares getName(); named like the receiver 'Department'
  - low `examples/userManagerAccessControl/model/beans/role.cfc` — declares getName()
  - low `tests/declared/things/myconfig.cfc` — declares getName()
  - low `tests/model/beans/user37c.cfc` — declares getName()
- `local.user → examples/userManagerAJAX/model/beans/department.cfc` — 1 finding(s), 6 candidate(s):
  - low `examples/userManagerAJAX/model/beans/department.cfc` — declares getName(); named like the receiver 'Department'
  - low `examples/userManager/model/beans/department.cfc` — declares getName(); named like the receiver 'Department'
  - low `examples/userManagerAccessControl/model/beans/department.cfc` — declares getName(); named like the receiver 'Department'
  - low `examples/userManagerAccessControl/model/beans/role.cfc` — declares getName()
  - low `tests/declared/things/myconfig.cfc` — declares getName()
  - low `tests/model/beans/user37c.cfc` — declares getName()
- `local.user → examples/userManagerAccessControl/model/beans/department.cfc` — 1 finding(s), 6 candidate(s):
  - low `examples/userManagerAccessControl/model/beans/department.cfc` — declares getName(); named like the receiver 'Department'
  - low `examples/userManager/model/beans/department.cfc` — declares getName(); named like the receiver 'Department'
  - low `examples/userManagerAJAX/model/beans/department.cfc` — declares getName(); named like the receiver 'Department'
  - low `examples/userManagerAccessControl/model/beans/role.cfc` — declares getName()
  - low `tests/declared/things/myconfig.cfc` — declares getName()
  - low `tests/model/beans/user37c.cfc` — declares getName()
- `rc.data → examples/qBall/model/beans/answer.cfc` — 1 finding(s), 14 candidate(s):
  - low `examples/qBall/model/beans/answer.cfc` — declares getId()
  - low `examples/qBall/model/beans/question.cfc` — declares getId()
  - low `examples/qBall/model/beans/user.cfc` — declares getId()
  - low `examples/userManager/model/beans/department.cfc` — declares getId()
  - low `examples/userManager/model/beans/user.cfc` — declares getId()
  - low `examples/userManagerAJAX/model/beans/department.cfc` — declares getId()
  - low `examples/userManagerAJAX/model/beans/user.cfc` — declares getId()
  - low `examples/userManagerAccessControl/model/beans/department.cfc` — declares getId()
  - … 6 more
- `rc.user → examples/qBall/model/beans/user.cfc` — 1 finding(s), 14 candidate(s):
  - low `examples/qBall/model/beans/user.cfc` — declares getId(); named like the receiver 'user'
  - low `examples/userManager/model/beans/user.cfc` — declares getId(); named like the receiver 'user'
  - low `examples/userManagerAJAX/model/beans/user.cfc` — declares getId(); named like the receiver 'user'
  - low `examples/userManagerAccessControl/model/beans/user.cfc` — declares getId(); named like the receiver 'user'
  - low `tests/services/user.cfc` — declares getId(); named like the receiver 'user'
  - low `examples/qBall/model/beans/answer.cfc` — declares getId()
  - low `examples/qBall/model/beans/question.cfc` — declares getId()
  - low `examples/userManager/model/beans/department.cfc` — declares getId()
  - … 6 more
- `variables.departmentservice → examples/userManagerAJAX/controllers/user.cfc` — 1 finding(s), 14 candidate(s):
  - low `examples/userManagerAJAX/controllers/user.cfc` — declares list()
  - low `examples/userManagerAJAX/model/services/department.cfc` — declares list()
  - low `examples/userManagerAJAX/model/services/user.cfc` — declares list()
  - low `examples/qBall/controllers/question.cfc` — declares list()
  - low `examples/todos/controllers/main.cfc` — declares list()
  - low `examples/userManager/controllers/user.cfc` — declares list()
  - low `examples/userManagerAccessControl/controllers/user.cfc` — declares list()
  - low `examples/qBall/model/services/question.cfc` — declares list()
  - … 6 more
- `variables.departmentservice → examples/userManagerAJAX/model/services/department.cfc` — 1 finding(s), 11 candidate(s):
  - low `examples/userManagerAJAX/model/services/department.cfc` — declares get()
  - low `examples/userManagerAJAX/model/services/user.cfc` — declares get()
  - low `examples/todos/controllers/main.cfc` — declares get()
  - low `examples/userManager/controllers/user.cfc` — declares get()
  - low `examples/qBall/model/services/user.cfc` — declares get()
  - low `examples/todos/model/services/todo.cfc` — declares get()
  - low `examples/userManager/model/services/department.cfc` — declares get()
  - low `examples/userManager/model/services/user.cfc` — declares get()
  - … 3 more
- `variables.departmentservice → examples/userManagerAccessControl/controllers/user.cfc` — 1 finding(s), 14 candidate(s):
  - low `examples/userManagerAccessControl/controllers/user.cfc` — declares list()
  - low `examples/userManagerAccessControl/model/services/department.cfc` — declares list()
  - low `examples/userManagerAccessControl/model/services/role.cfc` — declares list()
  - low `examples/userManagerAccessControl/model/services/user.cfc` — declares list()
  - low `examples/qBall/controllers/question.cfc` — declares list()
  - low `examples/todos/controllers/main.cfc` — declares list()
  - low `examples/userManager/controllers/user.cfc` — declares list()
  - low `examples/userManagerAJAX/controllers/user.cfc` — declares list()
  - … 6 more
- `variables.roleservice → examples/userManagerAccessControl/controllers/user.cfc` — 1 finding(s), 14 candidate(s):
  - low `examples/userManagerAccessControl/controllers/user.cfc` — declares list()
  - low `examples/userManagerAccessControl/model/services/department.cfc` — declares list()
  - low `examples/userManagerAccessControl/model/services/role.cfc` — declares list()
  - low `examples/userManagerAccessControl/model/services/user.cfc` — declares list()
  - low `examples/qBall/controllers/question.cfc` — declares list()
  - low `examples/todos/controllers/main.cfc` — declares list()
  - low `examples/userManager/controllers/user.cfc` — declares list()
  - low `examples/userManagerAJAX/controllers/user.cfc` — declares list()
  - … 6 more
- `variables.roleservice → examples/userManagerAccessControl/model/services/department.cfc` — 1 finding(s), 11 candidate(s):
  - low `examples/userManagerAccessControl/model/services/department.cfc` — declares get()
  - low `examples/userManagerAccessControl/model/services/role.cfc` — declares get()
  - low `examples/userManagerAccessControl/model/services/user.cfc` — declares get()
  - low `examples/todos/controllers/main.cfc` — declares get()
  - low `examples/userManager/controllers/user.cfc` — declares get()
  - low `examples/qBall/model/services/user.cfc` — declares get()
  - low `examples/todos/model/services/todo.cfc` — declares get()
  - low `examples/userManager/model/services/department.cfc` — declares get()
  - … 3 more
- `variables.user → examples/qBall/model/services/user.cfc` — 1 finding(s), 11 candidate(s):
  - low `examples/qBall/model/services/user.cfc` — declares get(); named like the receiver 'user'
  - low `examples/userManager/controllers/user.cfc` — declares get(); named like the receiver 'user'
  - low `examples/userManager/model/services/user.cfc` — declares get(); named like the receiver 'user'
  - low `examples/userManagerAJAX/model/services/user.cfc` — declares get(); named like the receiver 'user'
  - low `examples/userManagerAccessControl/model/services/user.cfc` — declares get(); named like the receiver 'user'
  - low `examples/todos/controllers/main.cfc` — declares get()
  - low `examples/todos/model/services/todo.cfc` — declares get()
  - low `examples/userManager/model/services/department.cfc` — declares get()
  - … 3 more
- `variables.userservice → examples/userManagerAJAX/model/services/department.cfc` — 1 finding(s), 11 candidate(s):
  - low `examples/userManagerAJAX/model/services/department.cfc` — declares get()
  - low `examples/userManagerAJAX/model/services/user.cfc` — declares get()
  - low `examples/todos/controllers/main.cfc` — declares get()
  - low `examples/userManager/controllers/user.cfc` — declares get()
  - low `examples/qBall/model/services/user.cfc` — declares get()
  - low `examples/todos/model/services/todo.cfc` — declares get()
  - low `examples/userManager/model/services/department.cfc` — declares get()
  - low `examples/userManager/model/services/user.cfc` — declares get()
  - … 3 more
- `variables.userservice → examples/userManagerAJAX/model/services/user.cfc` — 1 finding(s), 6 candidate(s):
  - low `examples/userManagerAJAX/model/services/user.cfc` — declares get(), save()
  - low `examples/todos/controllers/main.cfc` — declares get(), save()
  - low `examples/userManager/controllers/user.cfc` — declares get(), save()
  - low `examples/todos/model/services/todo.cfc` — declares get(), save()
  - low `examples/userManager/model/services/user.cfc` — declares get(), save()
  - low `examples/userManagerAccessControl/model/services/user.cfc` — declares get(), save()
- `variables.userservice → examples/userManagerAccessControl/model/services/department.cfc` — 1 finding(s), 11 candidate(s):
  - low `examples/userManagerAccessControl/model/services/department.cfc` — declares get()
  - low `examples/userManagerAccessControl/model/services/role.cfc` — declares get()
  - low `examples/userManagerAccessControl/model/services/user.cfc` — declares get()
  - low `examples/todos/controllers/main.cfc` — declares get()
  - low `examples/userManager/controllers/user.cfc` — declares get()
  - low `examples/qBall/model/services/user.cfc` — declares get()
  - low `examples/todos/model/services/todo.cfc` — declares get()
  - low `examples/userManager/model/services/department.cfc` — declares get()
  - … 3 more

</details>

## Return types — a call chained on a method that declares no component

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 15 | `method 'getStackLog' has no component return type → tests/issue518/stackLog.cfc` | 2 | medium | declares log(); named like the receiver 'StackLog' (2 candidates) | `tests/issue518/daos/advReverse.cfc:6` method 'getStackLog' has no component return type (chain to 'log') |
| 11 | `method 'getStackLogService' has no component return type → tests/aop/services/stackLogService.cfc` | 2 | medium | declares log(); named like the receiver 'StackLogService' (2 candidates) | `tests/aop/interceptors/aop/AfterInterceptor.cfc:10` method 'getStackLogService' has no component return type (chain to 'log') |
| 6 | `method '_get_framework_one' has no component return type → Application.cfc` | 6 | low | declares onApplicationStart() (6 candidates) | `Application.cfc:45` method '_get_framework_one' has no component return type (chain to 'onApplicationStart') |
| 6 | `method '_get_framework_one' has no component return type → examples/Application.cfc` | 6 | low | declares onApplicationStart() (6 candidates) | `examples/Application.cfc:47` method '_get_framework_one' has no component return type (chain to 'onApplicationStart') |
| 6 | `method '_get_framework_one' has no component return type → examples/rest/Application.cfc` | 6 | low | declares onApplicationStart() (6 candidates) | `examples/rest/Application.cfc:17` method '_get_framework_one' has no component return type (chain to 'onApplicationStart') |
| 6 | `method '_get_framework_one' has no component return type → examples/userManager/Application.cfc` | 6 | low | declares onApplicationStart() (6 candidates) | `examples/userManager/Application.cfc:41` method '_get_framework_one' has no component return type (chain to 'onApplicationStart') |
| 6 | `method '_get_framework_one' has no component return type → framework/Application.cfc` | 6 | low | declares onApplicationStart() (6 candidates) | `framework/Application.cfc:42` method '_get_framework_one' has no component return type (chain to 'onApplicationStart') |
| 2 | `method 'getDefaultBeanFactory' has no component return type → framework/WireBoxAdapter.cfc` | 2 | low | declares containsBean(), getBean() (2 candidates) | `framework/one.cfc:1849` method 'getDefaultBeanFactory' has no component return type (chain to 'containsBean') |
| 2 | `method 'getSubsystemBeanFactory' has no component return type → framework/WireBoxAdapter.cfc` | 2 | low | declares containsBean(), getBean() (2 candidates) | `framework/one.cfc:1847` method 'getSubsystemBeanFactory' has no component return type (chain to 'containsBean') |
| 1 | `method 'getBean' has no component return type → framework/ioc.cfc` | 2 | low | declares onLoad() (2 candidates) | `framework/ioc.cfc:687` method 'getBean' has no component return type (chain to 'onLoad') |
| 1 | `method 'getBeanFactory' has no component return type → framework/ioc.cfc` | 1 | medium | declares setParent() | `framework/one.cfc:1462` method 'getBeanFactory' has no component return type (chain to 'setParent') |
| 1 | `method 'getBeanFactory' in framework.one has no component return type → framework/WireBoxAdapter.cfc` | 2 | low | declares getBean() (2 candidates) | `examples/userManagerAccessControl/controllers/user.cfc:38` method 'getBeanFactory' in framework.one has no component return type (chain to 'getBean') |
| 1 | `method 'getUserService' has no component return type → examples/userManager/controllers/user.cfc` | 11 | low | declares get() (11 candidates) | `examples/userManager/controllers/user.cfc:34` method 'getUserService' has no component return type (chain to 'get') |
| 1 | `method 'getUserService' in product has no component return type → tests/services/user.cfc` | 14 | low | declares getId() (14 candidates) | `tests/ModelServiceTest.cfc:27` method 'getUserService' in product has no component return type (chain to 'getId') |

<details><summary>Groups with several candidates</summary>

- `method 'getStackLog' has no component return type → tests/issue518/stackLog.cfc` — 15 finding(s), 2 candidate(s):
  - medium `tests/issue518/stackLog.cfc` — declares log(); named like the receiver 'StackLog'
  - low `tests/aop/services/stackLogService.cfc` — declares log()
- `method 'getStackLogService' has no component return type → tests/aop/services/stackLogService.cfc` — 11 finding(s), 2 candidate(s):
  - medium `tests/aop/services/stackLogService.cfc` — declares log(); named like the receiver 'StackLogService'
  - low `tests/issue518/stackLog.cfc` — declares log()
- `method '_get_framework_one' has no component return type → Application.cfc` — 6 finding(s), 6 candidate(s):
  - low `Application.cfc` — declares onApplicationStart()
  - low `examples/Application.cfc` — declares onApplicationStart()
  - low `framework/Application.cfc` — declares onApplicationStart()
  - low `framework/one.cfc` — declares onApplicationStart()
  - low `examples/rest/Application.cfc` — declares onApplicationStart()
  - low `examples/userManager/Application.cfc` — declares onApplicationStart()
- `method '_get_framework_one' has no component return type → examples/Application.cfc` — 6 finding(s), 6 candidate(s):
  - low `examples/Application.cfc` — declares onApplicationStart()
  - low `Application.cfc` — declares onApplicationStart()
  - low `examples/rest/Application.cfc` — declares onApplicationStart()
  - low `examples/userManager/Application.cfc` — declares onApplicationStart()
  - low `framework/Application.cfc` — declares onApplicationStart()
  - low `framework/one.cfc` — declares onApplicationStart()
- `method '_get_framework_one' has no component return type → examples/rest/Application.cfc` — 6 finding(s), 6 candidate(s):
  - low `examples/rest/Application.cfc` — declares onApplicationStart()
  - low `examples/Application.cfc` — declares onApplicationStart()
  - low `Application.cfc` — declares onApplicationStart()
  - low `examples/userManager/Application.cfc` — declares onApplicationStart()
  - low `framework/Application.cfc` — declares onApplicationStart()
  - low `framework/one.cfc` — declares onApplicationStart()
- `method '_get_framework_one' has no component return type → examples/userManager/Application.cfc` — 6 finding(s), 6 candidate(s):
  - low `examples/userManager/Application.cfc` — declares onApplicationStart()
  - low `examples/Application.cfc` — declares onApplicationStart()
  - low `Application.cfc` — declares onApplicationStart()
  - low `examples/rest/Application.cfc` — declares onApplicationStart()
  - low `framework/Application.cfc` — declares onApplicationStart()
  - low `framework/one.cfc` — declares onApplicationStart()
- `method '_get_framework_one' has no component return type → framework/Application.cfc` — 6 finding(s), 6 candidate(s):
  - low `framework/Application.cfc` — declares onApplicationStart()
  - low `framework/one.cfc` — declares onApplicationStart()
  - low `Application.cfc` — declares onApplicationStart()
  - low `examples/Application.cfc` — declares onApplicationStart()
  - low `examples/rest/Application.cfc` — declares onApplicationStart()
  - low `examples/userManager/Application.cfc` — declares onApplicationStart()
- `method 'getDefaultBeanFactory' has no component return type → framework/WireBoxAdapter.cfc` — 2 finding(s), 2 candidate(s):
  - low `framework/WireBoxAdapter.cfc` — declares containsBean(), getBean()
  - low `framework/ioc.cfc` — declares containsBean(), getBean()
- `method 'getSubsystemBeanFactory' has no component return type → framework/WireBoxAdapter.cfc` — 2 finding(s), 2 candidate(s):
  - low `framework/WireBoxAdapter.cfc` — declares containsBean(), getBean()
  - low `framework/ioc.cfc` — declares containsBean(), getBean()
- `method 'getBean' has no component return type → framework/ioc.cfc` — 1 finding(s), 2 candidate(s):
  - low `framework/ioc.cfc` — declares onLoad()
  - low `tests/model/services/listener.cfc` — declares onLoad()
- `method 'getBeanFactory' in framework.one has no component return type → framework/WireBoxAdapter.cfc` — 1 finding(s), 2 candidate(s):
  - low `framework/WireBoxAdapter.cfc` — declares getBean()
  - low `framework/ioc.cfc` — declares getBean()
- `method 'getUserService' has no component return type → examples/userManager/controllers/user.cfc` — 1 finding(s), 11 candidate(s):
  - low `examples/userManager/controllers/user.cfc` — declares get()
  - low `examples/userManager/model/services/department.cfc` — declares get()
  - low `examples/userManager/model/services/user.cfc` — declares get()
  - low `examples/todos/controllers/main.cfc` — declares get()
  - low `examples/qBall/model/services/user.cfc` — declares get()
  - low `examples/todos/model/services/todo.cfc` — declares get()
  - low `examples/userManagerAJAX/model/services/department.cfc` — declares get()
  - low `examples/userManagerAJAX/model/services/user.cfc` — declares get()
  - … 3 more
- `method 'getUserService' in product has no component return type → tests/services/user.cfc` — 1 finding(s), 14 candidate(s):
  - low `tests/services/user.cfc` — declares getId()
  - low `tests/model/beans/user37.cfc` — declares getId()
  - low `tests/model/beans/user37b.cfc` — declares getId()
  - low `tests/model/beans/user37c.cfc` — declares getId()
  - low `examples/qBall/model/beans/answer.cfc` — declares getId()
  - low `examples/qBall/model/beans/question.cfc` — declares getId()
  - low `examples/qBall/model/beans/user.cfc` — declares getId()
  - low `examples/userManager/model/beans/department.cfc` — declares getId()
  - … 6 more

</details>

## Method definitions — a method not found where it was looked for

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 3 | `buildurl` | 1 | medium | declares buildURL | `examples/modular/pages/main/default.cfm:4` no qualifier, not in file |
| 3 | `proceed` | 0 | none |  | `framework/beanProxy.cfc:231` no qualifier, not in file |
| 2 | `itemtest` | 1 | medium | declares itemTest | `tests/ExtraBeansTest.cfc:40` method 'itemTest' not found in user |
| 2 | `setlayout` | 1 | medium | declares setLayout | `tests/onMissingViewLayoutTest.cfc:56` not found in extends chain |
| 2 | `translateargs` | 0 | none |  | `tests/aop/interceptors/aop/BeforeInterceptor.cfc:12` not found in extends chain |
| 2 | `view` | 2 | low | declares view (2 candidates) | `tests/onMissingViewLayoutTest.cfc:57` not found in extends chain |
| 1 | `around` | 2 | low | declares around (2 candidates) | `framework/beanProxy.cfc:227` no qualifier, not in file |
| 1 | `callback` | 0 | none |  | `framework/one.cfc:1278` no qualifier, not in file |
| 1 | `getinterceptedmethods` | 0 | none |  | `framework/beanProxy.cfc:222` no qualifier, not in file |
| 1 | `getitemlamb` | 1 | medium | declares getItemLamb | `tests/ExtraBeansTest.cfc:43` method 'getItemLamb' not found in user |
| 1 | `getproduct` | 1 | medium | declares getProduct | `tests/ModelTest.cfc:36` method 'getProduct' not found in user |
| 1 | `islast` | 0 | none |  | `framework/beanProxy.cfc:241` no qualifier, not in file |

<details><summary>Groups with several candidates</summary>

- `view` — 2 finding(s), 2 candidate(s):
  - low `framework/one.cfc:1414` — declares view
  - low `examples/qBall/controllers/question.cfc:77` — declares view
- `around` — 1 finding(s), 2 candidate(s):
  - low `tests/aop/interceptors/aop/AroundInterceptor.cfc:9` — declares around
  - low `tests/issue518/interceptors/aop/AroundInterceptor.cfc:9` — declares around

</details>

## Object references — a component path that names no file

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 1 | `base component does not resolve; 3 inherited calls not checked` | ? | none |  | `framework/WireBoxAdapter.cfc:1` base component does not resolve; 3 inherited calls not checked |
