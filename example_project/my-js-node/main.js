let string1 = GetValue("string1") || "";
let reversed = string1.split("").reverse().join("");
SetValue("string2", reversed);
