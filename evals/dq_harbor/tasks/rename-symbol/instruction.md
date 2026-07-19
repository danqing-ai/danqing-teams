Under `/app/pkg/` is a tiny Python package. Rename the function `old_name` to `new_name` **everywhere**
(definition and all call sites / imports that refer to that function name as an identifier).

Do not rename unrelated identifiers that merely contain the substring. Keep behavior the same otherwise.
After refactor, `python3 -c 'from pkg.api import use; print(use())'` must print `42`.

Do not ask the user any questions.
