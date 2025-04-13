"use strict";
var bangumiWikiParser = (() => {
  var __defProp = Object.defineProperty;
  var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __hasOwnProp = Object.prototype.hasOwnProperty;
  var __export = (target, all) => {
    for (var name in all)
      __defProp(target, name, { get: all[name], enumerable: true });
  };
  var __copyProps = (to, from, except, desc) => {
    if (from && typeof from === "object" || typeof from === "function") {
      for (let key of __getOwnPropNames(from))
        if (!__hasOwnProp.call(to, key) && key !== except)
          __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
    }
    return to;
  };
  var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

  // src/index.ts
  var src_exports = {};
  __export(src_exports, {
    ArrayItemWrappedError: () => ArrayItemWrappedError,
    ArrayNoCloseError: () => ArrayNoCloseError,
    ExpectingNewFieldError: () => ExpectingNewFieldError,
    ExpectingSignEqualError: () => ExpectingSignEqualError,
    GlobalPrefixError: () => GlobalPrefixError,
    GlobalSuffixError: () => GlobalSuffixError,
    WikiArrayItem: () => WikiArrayItem,
    WikiItem: () => WikiItem,
    WikiSyntaxError: () => WikiSyntaxError,
    parse: () => parse,
    parse2: () => parse2,
    parseToMap: () => parseToMap,
    parseToMap2: () => parseToMap2,
    stringify: () => stringify,
    stringifyMap: () => stringifyMap
  });

  // src/error.ts
  var GlobalPrefixError = "missing prefix '{{Infobox' at the start";
  var GlobalSuffixError = "missing suffix '}}' at the end";
  var ArrayNoCloseError = "array should be closed by '}'";
  var ArrayItemWrappedError = "array item should be wrapped by '[]'";
  var ExpectingNewFieldError = "missing '|' to start a new field";
  var ExpectingSignEqualError = "missing '=' to separate field name and value";
  var WikiSyntaxError = class extends Error {
    lino;
    line;
    constructor(lino, line, message) {
      super(toErrorString(lino, line, message));
      this.line = line;
      this.lino = lino;
    }
  };
  function toErrorString(lino, line, msg) {
    if (line === null) {
      return `WikiSyntaxError: ${msg}, line ${lino}`;
    }
    return `WikiSyntaxError: ${msg}, line ${lino}: ${JSON.stringify(line)}`;
  }

  // src/shared.ts
  var prefix = "{{Infobox";
  var suffix = "}}";

  // src/types.ts
  var WikiArrayItem = class {
    k;
    v;
    constructor(k, v) {
      this.k = k || void 0;
      this.v = v;
    }
  };
  var WikiItem = class {
    key;
    value;
    array;
    values;
    constructor(key, value, type) {
      this.key = key;
      switch (type) {
        case "array": {
          this.array = true;
          this.values = [];
          break;
        }
        case "object": {
          this.value = value;
          break;
        }
      }
    }
    convertToArray() {
      if (this.array) {
        return;
      }
      this.array = true;
      this.values ??= [];
      this.values.push(new WikiArrayItem(void 0, this.value));
      this.value = void 0;
    }
    push(item) {
      this.convertToArray();
      if (item.array) {
        this.values?.push(...item.values ?? []);
      } else {
        this.values?.push(new WikiArrayItem(void 0, item.value));
      }
    }
  };

  // src/stringify.ts
  var stringifyArray = (arr) => {
    if (!arr) {
      return "";
    }
    return arr.reduce((pre, item) => `${pre}
[${item.k ? `${item.k}|` : ""}${item.v ?? ""}]`, "");
  };
  function stringify(wiki) {
    const body = wiki.data.reduce((pre, item) => {
      if (item.array) {
        return `${pre}
|${item.key} = {${stringifyArray(item.values)}
}`;
      }
      return `${pre}
|${item.key} = ${item.value ?? ""}`;
    }, "");
    const type = wiki.type ? " " + wiki.type : "";
    return `${prefix}${type}${body}
${suffix}`;
  }
  function stringifyMap(wiki) {
    const body = [...wiki.data].map(([key, value]) => {
      if (typeof value === "string") {
        return `|${key} = ${value ?? ""}`;
      }
      return `|${key} = {${stringifyArray(value)}
}`;
    }).join("\n");
    const type = wiki.type ? " " + wiki.type : "";
    return `${prefix}${type}
${body}
${suffix}`;
  }

  // src/index.ts
  function parseToMap2(s) {
    try {
      return [null, parseToMap(s)];
    } catch (error) {
      if (error instanceof WikiSyntaxError) {
        return [error, null];
      }
      throw error;
    }
  }
  function parseToMap(s) {
    const w = parse(s);
    const data = /* @__PURE__ */ new Map();
    for (const item of w.data) {
      let previous = data.get(item.key);
      if (previous) {
        if (typeof previous === "string") {
          previous = [new WikiArrayItem(void 0, previous)];
        }
        if (item.array) {
          previous.push(...item.values);
        } else {
          previous.push(new WikiArrayItem(void 0, item.value));
        }
        data.set(item.key, previous);
        continue;
      }
      if (item.array) {
        data.set(item.key, item.values ?? []);
      } else {
        data.set(item.key, item.value);
      }
    }
    return { type: w.type, data };
  }
  function processInput(s) {
    let offset = 2;
    s = s.replaceAll("\r\n", "\n");
    for (const char of s) {
      switch (char) {
        case "\n": {
          offset++;
          break;
        }
        case " ":
        case "	": {
          continue;
        }
        default: {
          return [s.trim(), offset];
        }
      }
    }
    return [s.trim(), offset];
  }
  function parse2(s) {
    try {
      return [null, parse(s)];
    } catch (error) {
      if (error instanceof WikiSyntaxError) {
        return [error, null];
      }
      throw error;
    }
  }
  function parse(s) {
    const wiki = {
      type: "",
      data: []
    };
    const [strTrim, offset] = processInput(s);
    if (strTrim === "") {
      return wiki;
    }
    if (!strTrim.startsWith(prefix)) {
      throw new WikiSyntaxError(offset - 1, null, GlobalPrefixError);
    }
    if (!strTrim.endsWith(suffix)) {
      throw new WikiSyntaxError((s.match(/\n/g)?.length ?? -2) + 1, null, GlobalSuffixError);
    }
    const arr = strTrim.split("\n");
    if (arr[0]) {
      wiki.type = parseType(arr[0]);
    }
    const fields = arr.slice(1, -1);
    let inArray = false;
    for (let i = 0; i < fields.length; ++i) {
      const line = fields[i]?.trim();
      const lino = offset + i;
      if (!line) {
        continue;
      }
      if (line.startsWith("|")) {
        if (inArray) {
          throw new WikiSyntaxError(lino, line, ArrayNoCloseError);
        }
        const meta = parseNewField(lino, line);
        inArray = meta[2] === "array";
        const field = new WikiItem(...meta);
        wiki.data.push(field);
      } else if (inArray) {
        if (line.startsWith("}")) {
          inArray = false;
          continue;
        }
        if (i === fields.length - 1) {
          throw new WikiSyntaxError(lino, line, ArrayNoCloseError);
        }
        wiki.data.at(-1)?.values?.push(new WikiArrayItem(...parseArrayItem(lino, line)));
      } else {
        throw new WikiSyntaxError(lino, line, ExpectingNewFieldError);
      }
    }
    return wiki;
  }
  var parseType = (line) => {
    if (!line.includes("}}")) {
      return line.slice(prefix.length).trim();
    }
    return line.slice(prefix.length, line.indexOf("}}")).trim();
  };
  var parseNewField = (lino, line) => {
    const str = line.slice(1);
    const index = str.indexOf("=");
    if (index === -1) {
      throw new WikiSyntaxError(lino, line, ExpectingSignEqualError);
    }
    const key = str.slice(0, index).trim();
    const value = str.slice(index + 1).trim();
    switch (value) {
      case "{": {
        return [key, "", "array"];
      }
      default: {
        return [key, value, "object"];
      }
    }
  };
  var parseArrayItem = (lino, line) => {
    if (!line.startsWith("[") || !line.endsWith("]")) {
      throw new WikiSyntaxError(lino, line, ArrayItemWrappedError);
    }
    const content = line.slice(1, -1);
    const index = content.indexOf("|");
    if (index === -1) {
      return ["", content.trim()];
    }
    return [content.slice(0, index).trim(), content.slice(index + 1).trim()];
  };
  return __toCommonJS(src_exports);
})();
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsiLi4vc3JjL2luZGV4LnRzIiwgIi4uL3NyYy9lcnJvci50cyIsICIuLi9zcmMvc2hhcmVkLnRzIiwgIi4uL3NyYy90eXBlcy50cyIsICIuLi9zcmMvc3RyaW5naWZ5LnRzIl0sCiAgInNvdXJjZXNDb250ZW50IjogWyJpbXBvcnQge1xuICBBcnJheUl0ZW1XcmFwcGVkRXJyb3IsXG4gIEFycmF5Tm9DbG9zZUVycm9yLFxuICBFeHBlY3RpbmdOZXdGaWVsZEVycm9yLFxuICBFeHBlY3RpbmdTaWduRXF1YWxFcnJvcixcbiAgR2xvYmFsUHJlZml4RXJyb3IsXG4gIEdsb2JhbFN1ZmZpeEVycm9yLFxuICBXaWtpU3ludGF4RXJyb3IsXG59IGZyb20gJy4vZXJyb3IuanMnO1xuaW1wb3J0IHsgcHJlZml4LCBzdWZmaXggfSBmcm9tICcuL3NoYXJlZC5qcyc7XG5pbXBvcnQgdHlwZSB7IFdpa2ksIFdpa2lJdGVtVHlwZSwgV2lraU1hcCB9IGZyb20gJy4vdHlwZXMuanMnO1xuaW1wb3J0IHsgV2lraUFycmF5SXRlbSwgV2lraUl0ZW0gfSBmcm9tICcuL3R5cGVzLmpzJztcblxuZXhwb3J0ICogZnJvbSAnLi90eXBlcy5qcyc7XG5leHBvcnQgKiBmcm9tICcuL2Vycm9yLmpzJztcbmV4cG9ydCB7IHN0cmluZ2lmeSwgc3RyaW5naWZ5TWFwIH0gZnJvbSAnLi9zdHJpbmdpZnkuanMnO1xuXG5leHBvcnQgZnVuY3Rpb24gcGFyc2VUb01hcDIoczogc3RyaW5nKTogW251bGwsIFdpa2lNYXBdIHwgW1dpa2lTeW50YXhFcnJvciwgbnVsbF0ge1xuICB0cnkge1xuICAgIHJldHVybiBbbnVsbCwgcGFyc2VUb01hcChzKV07XG4gIH0gY2F0Y2ggKGVycm9yKSB7XG4gICAgaWYgKGVycm9yIGluc3RhbmNlb2YgV2lraVN5bnRheEVycm9yKSB7XG4gICAgICByZXR1cm4gW2Vycm9yLCBudWxsXTtcbiAgICB9XG5cbiAgICB0aHJvdyBlcnJvcjtcbiAgfVxufVxuXG4vKiogXHU4OUUzXHU2NzkwIHdpa2kgXHU2NTg3XHU2NzJDXHVGRjBDXHU0RUU1IGBNYXBgIFx1N0M3Qlx1NTc4Qlx1OEZENFx1NTZERVx1ODlFM1x1Njc5MFx1N0VEM1x1Njc5Q1x1MzAwMiBcdTRGMUFcdTU0MDhcdTVFNzZcdTkxQ0RcdTU5MERcdTUxRkFcdTczQjBcdTc2ODQga2V5ICovXG5leHBvcnQgZnVuY3Rpb24gcGFyc2VUb01hcChzOiBzdHJpbmcpOiBXaWtpTWFwIHtcbiAgY29uc3QgdyA9IHBhcnNlKHMpO1xuXG4gIGNvbnN0IGRhdGEgPSBuZXcgTWFwPHN0cmluZywgc3RyaW5nIHwgV2lraUFycmF5SXRlbVtdPigpO1xuXG4gIGZvciAoY29uc3QgaXRlbSBvZiB3LmRhdGEpIHtcbiAgICBsZXQgcHJldmlvdXMgPSBkYXRhLmdldChpdGVtLmtleSk7XG4gICAgaWYgKHByZXZpb3VzKSB7XG4gICAgICBpZiAodHlwZW9mIHByZXZpb3VzID09PSAnc3RyaW5nJykge1xuICAgICAgICBwcmV2aW91cyA9IFtuZXcgV2lraUFycmF5SXRlbSh1bmRlZmluZWQsIHByZXZpb3VzKV07XG4gICAgICB9XG4gICAgICBpZiAoaXRlbS5hcnJheSkge1xuICAgICAgICBwcmV2aW91cy5wdXNoKC4uLihpdGVtLnZhbHVlcyBhcyBXaWtpQXJyYXlJdGVtW10pKTtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIHByZXZpb3VzLnB1c2gobmV3IFdpa2lBcnJheUl0ZW0odW5kZWZpbmVkLCBpdGVtLnZhbHVlIGFzIHN0cmluZykpO1xuICAgICAgfVxuICAgICAgZGF0YS5zZXQoaXRlbS5rZXksIHByZXZpb3VzKTtcbiAgICAgIGNvbnRpbnVlO1xuICAgIH1cblxuICAgIGlmIChpdGVtLmFycmF5KSB7XG4gICAgICBkYXRhLnNldChpdGVtLmtleSwgaXRlbS52YWx1ZXMgPz8gW10pO1xuICAgIH0gZWxzZSB7XG4gICAgICBkYXRhLnNldChpdGVtLmtleSwgaXRlbS52YWx1ZSBhcyBzdHJpbmcpO1xuICAgIH1cbiAgfVxuXG4gIHJldHVybiB7IHR5cGU6IHcudHlwZSwgZGF0YSB9O1xufVxuXG5mdW5jdGlvbiBwcm9jZXNzSW5wdXQoczogc3RyaW5nKTogW3N0cmluZywgbnVtYmVyXSB7XG4gIGxldCBvZmZzZXQgPSAyO1xuICBzID0gcy5yZXBsYWNlQWxsKCdcXHJcXG4nLCAnXFxuJyk7XG5cbiAgZm9yIChjb25zdCBjaGFyIG9mIHMpIHtcbiAgICBzd2l0Y2ggKGNoYXIpIHtcbiAgICAgIGNhc2UgJ1xcbic6IHtcbiAgICAgICAgb2Zmc2V0Kys7XG4gICAgICAgIGJyZWFrO1xuICAgICAgfVxuICAgICAgY2FzZSAnICc6XG4gICAgICBjYXNlICdcXHQnOiB7XG4gICAgICAgIGNvbnRpbnVlO1xuICAgICAgfVxuICAgICAgZGVmYXVsdDoge1xuICAgICAgICByZXR1cm4gW3MudHJpbSgpLCBvZmZzZXRdO1xuICAgICAgfVxuICAgIH1cbiAgfVxuXG4gIHJldHVybiBbcy50cmltKCksIG9mZnNldF07XG59XG5cbmV4cG9ydCBmdW5jdGlvbiBwYXJzZTIoczogc3RyaW5nKTogW251bGwsIFdpa2ldIHwgW1dpa2lTeW50YXhFcnJvciwgbnVsbF0ge1xuICB0cnkge1xuICAgIHJldHVybiBbbnVsbCwgcGFyc2UocyldO1xuICB9IGNhdGNoIChlcnJvcikge1xuICAgIGlmIChlcnJvciBpbnN0YW5jZW9mIFdpa2lTeW50YXhFcnJvcikge1xuICAgICAgcmV0dXJuIFtlcnJvciwgbnVsbF07XG4gICAgfVxuXG4gICAgdGhyb3cgZXJyb3I7XG4gIH1cbn1cblxuZXhwb3J0IGZ1bmN0aW9uIHBhcnNlKHM6IHN0cmluZyk6IFdpa2kge1xuICBjb25zdCB3aWtpOiBXaWtpID0ge1xuICAgIHR5cGU6ICcnLFxuICAgIGRhdGE6IFtdLFxuICB9O1xuXG4gIGNvbnN0IFtzdHJUcmltLCBvZmZzZXRdID0gcHJvY2Vzc0lucHV0KHMpO1xuXG4gIGlmIChzdHJUcmltID09PSAnJykge1xuICAgIHJldHVybiB3aWtpO1xuICB9XG5cbiAgaWYgKCFzdHJUcmltLnN0YXJ0c1dpdGgocHJlZml4KSkge1xuICAgIHRocm93IG5ldyBXaWtpU3ludGF4RXJyb3Iob2Zmc2V0IC0gMSwgbnVsbCwgR2xvYmFsUHJlZml4RXJyb3IpO1xuICB9XG5cbiAgaWYgKCFzdHJUcmltLmVuZHNXaXRoKHN1ZmZpeCkpIHtcbiAgICB0aHJvdyBuZXcgV2lraVN5bnRheEVycm9yKChzLm1hdGNoKC9cXG4vZyk/Lmxlbmd0aCA/PyAtMikgKyAxLCBudWxsLCBHbG9iYWxTdWZmaXhFcnJvcik7XG4gIH1cblxuICBjb25zdCBhcnIgPSBzdHJUcmltLnNwbGl0KCdcXG4nKTtcbiAgaWYgKGFyclswXSkge1xuICAgIHdpa2kudHlwZSA9IHBhcnNlVHlwZShhcnJbMF0pO1xuICB9XG5cbiAgLyogc3BsaXQgY29udGVudCBiZXR3ZWVuIHt7SW5mb2JveCB4eHggYW5kIH19ICovXG4gIGNvbnN0IGZpZWxkcyA9IGFyci5zbGljZSgxLCAtMSk7XG5cbiAgbGV0IGluQXJyYXkgPSBmYWxzZTtcbiAgZm9yIChsZXQgaSA9IDA7IGkgPCBmaWVsZHMubGVuZ3RoOyArK2kpIHtcbiAgICBjb25zdCBsaW5lID0gZmllbGRzW2ldPy50cmltKCk7XG4gICAgY29uc3QgbGlubyA9IG9mZnNldCArIGk7XG5cbiAgICBpZiAoIWxpbmUpIHtcbiAgICAgIGNvbnRpbnVlO1xuICAgIH1cbiAgICAvKiBuZXcgZmllbGQgKi9cbiAgICBpZiAobGluZS5zdGFydHNXaXRoKCd8JykpIHtcbiAgICAgIGlmIChpbkFycmF5KSB7XG4gICAgICAgIHRocm93IG5ldyBXaWtpU3ludGF4RXJyb3IobGlubywgbGluZSwgQXJyYXlOb0Nsb3NlRXJyb3IpO1xuICAgICAgfVxuICAgICAgY29uc3QgbWV0YSA9IHBhcnNlTmV3RmllbGQobGlubywgbGluZSk7XG4gICAgICBpbkFycmF5ID0gbWV0YVsyXSA9PT0gJ2FycmF5JztcbiAgICAgIGNvbnN0IGZpZWxkID0gbmV3IFdpa2lJdGVtKC4uLm1ldGEpO1xuICAgICAgd2lraS5kYXRhLnB1c2goZmllbGQpO1xuICAgICAgLyogaXMgQXJyYXkgaXRlbSAqL1xuICAgIH0gZWxzZSBpZiAoaW5BcnJheSkge1xuICAgICAgaWYgKGxpbmUuc3RhcnRzV2l0aCgnfScpKSB7XG4gICAgICAgIGluQXJyYXkgPSBmYWxzZTtcbiAgICAgICAgY29udGludWU7XG4gICAgICB9XG4gICAgICBpZiAoaSA9PT0gZmllbGRzLmxlbmd0aCAtIDEpIHtcbiAgICAgICAgdGhyb3cgbmV3IFdpa2lTeW50YXhFcnJvcihsaW5vLCBsaW5lLCBBcnJheU5vQ2xvc2VFcnJvcik7XG4gICAgICB9XG4gICAgICB3aWtpLmRhdGEuYXQoLTEpPy52YWx1ZXM/LnB1c2gobmV3IFdpa2lBcnJheUl0ZW0oLi4ucGFyc2VBcnJheUl0ZW0obGlubywgbGluZSkpKTtcbiAgICB9IGVsc2Uge1xuICAgICAgdGhyb3cgbmV3IFdpa2lTeW50YXhFcnJvcihsaW5vLCBsaW5lLCBFeHBlY3RpbmdOZXdGaWVsZEVycm9yKTtcbiAgICB9XG4gIH1cbiAgcmV0dXJuIHdpa2k7XG59XG5cbmNvbnN0IHBhcnNlVHlwZSA9IChsaW5lOiBzdHJpbmcpOiBzdHJpbmcgPT4ge1xuICBpZiAoIWxpbmUuaW5jbHVkZXMoJ319JykpIHtcbiAgICByZXR1cm4gbGluZS5zbGljZShwcmVmaXgubGVuZ3RoKS50cmltKCk7XG4gIH1cbiAgcmV0dXJuIGxpbmUuc2xpY2UocHJlZml4Lmxlbmd0aCwgbGluZS5pbmRleE9mKCd9fScpKS50cmltKCk7XG59O1xuXG5jb25zdCBwYXJzZU5ld0ZpZWxkID0gKGxpbm86IG51bWJlciwgbGluZTogc3RyaW5nKTogW3N0cmluZywgc3RyaW5nLCBXaWtpSXRlbVR5cGVdID0+IHtcbiAgY29uc3Qgc3RyID0gbGluZS5zbGljZSgxKTtcbiAgY29uc3QgaW5kZXggPSBzdHIuaW5kZXhPZignPScpO1xuXG4gIGlmIChpbmRleCA9PT0gLTEpIHtcbiAgICB0aHJvdyBuZXcgV2lraVN5bnRheEVycm9yKGxpbm8sIGxpbmUsIEV4cGVjdGluZ1NpZ25FcXVhbEVycm9yKTtcbiAgfVxuXG4gIGNvbnN0IGtleSA9IHN0ci5zbGljZSgwLCBpbmRleCkudHJpbSgpO1xuICBjb25zdCB2YWx1ZSA9IHN0ci5zbGljZShpbmRleCArIDEpLnRyaW0oKTtcbiAgc3dpdGNoICh2YWx1ZSkge1xuICAgIGNhc2UgJ3snOiB7XG4gICAgICByZXR1cm4gW2tleSwgJycsICdhcnJheSddO1xuICAgIH1cbiAgICBkZWZhdWx0OiB7XG4gICAgICByZXR1cm4gW2tleSwgdmFsdWUsICdvYmplY3QnXTtcbiAgICB9XG4gIH1cbn07XG5cbmNvbnN0IHBhcnNlQXJyYXlJdGVtID0gKGxpbm86IG51bWJlciwgbGluZTogc3RyaW5nKTogW3N0cmluZywgc3RyaW5nXSA9PiB7XG4gIGlmICghbGluZS5zdGFydHNXaXRoKCdbJykgfHwgIWxpbmUuZW5kc1dpdGgoJ10nKSkge1xuICAgIHRocm93IG5ldyBXaWtpU3ludGF4RXJyb3IobGlubywgbGluZSwgQXJyYXlJdGVtV3JhcHBlZEVycm9yKTtcbiAgfVxuICBjb25zdCBjb250ZW50ID0gbGluZS5zbGljZSgxLCAtMSk7XG4gIGNvbnN0IGluZGV4ID0gY29udGVudC5pbmRleE9mKCd8Jyk7XG4gIGlmIChpbmRleCA9PT0gLTEpIHtcbiAgICByZXR1cm4gWycnLCBjb250ZW50LnRyaW0oKV07XG4gIH1cbiAgcmV0dXJuIFtjb250ZW50LnNsaWNlKDAsIGluZGV4KS50cmltKCksIGNvbnRlbnQuc2xpY2UoaW5kZXggKyAxKS50cmltKCldO1xufTtcbiIsICJleHBvcnQgY29uc3QgR2xvYmFsUHJlZml4RXJyb3IgPSBcIm1pc3NpbmcgcHJlZml4ICd7e0luZm9ib3gnIGF0IHRoZSBzdGFydFwiO1xuZXhwb3J0IGNvbnN0IEdsb2JhbFN1ZmZpeEVycm9yID0gXCJtaXNzaW5nIHN1ZmZpeCAnfX0nIGF0IHRoZSBlbmRcIjtcbmV4cG9ydCBjb25zdCBBcnJheU5vQ2xvc2VFcnJvciA9IFwiYXJyYXkgc2hvdWxkIGJlIGNsb3NlZCBieSAnfSdcIjtcbmV4cG9ydCBjb25zdCBBcnJheUl0ZW1XcmFwcGVkRXJyb3IgPSBcImFycmF5IGl0ZW0gc2hvdWxkIGJlIHdyYXBwZWQgYnkgJ1tdJ1wiO1xuZXhwb3J0IGNvbnN0IEV4cGVjdGluZ05ld0ZpZWxkRXJyb3IgPSBcIm1pc3NpbmcgJ3wnIHRvIHN0YXJ0IGEgbmV3IGZpZWxkXCI7XG5leHBvcnQgY29uc3QgRXhwZWN0aW5nU2lnbkVxdWFsRXJyb3IgPSBcIm1pc3NpbmcgJz0nIHRvIHNlcGFyYXRlIGZpZWxkIG5hbWUgYW5kIHZhbHVlXCI7XG5cbmV4cG9ydCBjbGFzcyBXaWtpU3ludGF4RXJyb3IgZXh0ZW5kcyBFcnJvciB7XG4gIGxpbm86IG51bWJlcjtcbiAgbGluZTogc3RyaW5nIHwgbnVsbDtcblxuICBjb25zdHJ1Y3RvcihsaW5vOiBudW1iZXIsIGxpbmU6IHN0cmluZyB8IG51bGwsIG1lc3NhZ2U6IHN0cmluZykge1xuICAgIHN1cGVyKHRvRXJyb3JTdHJpbmcobGlubywgbGluZSwgbWVzc2FnZSkpO1xuICAgIHRoaXMubGluZSA9IGxpbmU7XG4gICAgdGhpcy5saW5vID0gbGlubztcbiAgfVxufVxuXG5mdW5jdGlvbiB0b0Vycm9yU3RyaW5nKGxpbm86IG51bWJlciwgbGluZTogc3RyaW5nIHwgbnVsbCwgbXNnOiBzdHJpbmcpOiBzdHJpbmcge1xuICBpZiAobGluZSA9PT0gbnVsbCkge1xuICAgIHJldHVybiBgV2lraVN5bnRheEVycm9yOiAke21zZ30sIGxpbmUgJHtsaW5vfWA7XG4gIH1cblxuICByZXR1cm4gYFdpa2lTeW50YXhFcnJvcjogJHttc2d9LCBsaW5lICR7bGlub306ICR7SlNPTi5zdHJpbmdpZnkobGluZSl9YDtcbn1cbiIsICIvKiBzaG91bGQgc3RhcnQgd2l0aCBge3tJbmZvYm94YCBhbmQgZW5kIHdpdGggYH19YCAqL1xuZXhwb3J0IGNvbnN0IHByZWZpeCA9ICd7e0luZm9ib3gnO1xuZXhwb3J0IGNvbnN0IHN1ZmZpeCA9ICd9fSc7XG4iLCAiZXhwb3J0IGludGVyZmFjZSBXaWtpIHtcbiAgdHlwZTogc3RyaW5nO1xuICBkYXRhOiBXaWtpSXRlbVtdO1xufVxuXG4vKiogSlMgXHU3Njg0IG1hcCBcdTRGMUFcdTYzMDlcdTcxNjdcdTYzRDJcdTUxNjVcdTk4N0FcdTVFOEZcdTYzOTJcdTVFOEYgKi9cbmV4cG9ydCBpbnRlcmZhY2UgV2lraU1hcCB7XG4gIHR5cGU6IHN0cmluZztcbiAgZGF0YTogTWFwPHN0cmluZywgc3RyaW5nIHwgV2lraUFycmF5SXRlbVtdPjtcbn1cblxuZXhwb3J0IHR5cGUgV2lraUl0ZW1UeXBlID0gJ2FycmF5JyB8ICdvYmplY3QnO1xuXG5leHBvcnQgY2xhc3MgV2lraUFycmF5SXRlbSB7XG4gIGs/OiBzdHJpbmc7XG4gIHY/OiBzdHJpbmc7XG5cbiAgY29uc3RydWN0b3Ioaz86IHN0cmluZywgdj86IHN0cmluZykge1xuICAgIHRoaXMuayA9IGsgfHwgdW5kZWZpbmVkO1xuICAgIHRoaXMudiA9IHY7XG4gIH1cbn1cblxuZXhwb3J0IGNsYXNzIFdpa2lJdGVtIHtcbiAga2V5OiBzdHJpbmc7XG4gIHZhbHVlPzogc3RyaW5nO1xuICBhcnJheT86IGJvb2xlYW47XG4gIHZhbHVlcz86IFdpa2lBcnJheUl0ZW1bXTtcblxuICBjb25zdHJ1Y3RvcihrZXk6IHN0cmluZywgdmFsdWU6IHN0cmluZywgdHlwZTogV2lraUl0ZW1UeXBlKSB7XG4gICAgdGhpcy5rZXkgPSBrZXk7XG4gICAgc3dpdGNoICh0eXBlKSB7XG4gICAgICBjYXNlICdhcnJheSc6IHtcbiAgICAgICAgdGhpcy5hcnJheSA9IHRydWU7XG4gICAgICAgIHRoaXMudmFsdWVzID0gW107XG4gICAgICAgIGJyZWFrO1xuICAgICAgfVxuICAgICAgY2FzZSAnb2JqZWN0Jzoge1xuICAgICAgICB0aGlzLnZhbHVlID0gdmFsdWU7XG4gICAgICAgIGJyZWFrO1xuICAgICAgfVxuICAgIH1cbiAgfVxuXG4gIHByaXZhdGUgY29udmVydFRvQXJyYXkoKSB7XG4gICAgaWYgKHRoaXMuYXJyYXkpIHtcbiAgICAgIHJldHVybjtcbiAgICB9XG5cbiAgICB0aGlzLmFycmF5ID0gdHJ1ZTtcbiAgICB0aGlzLnZhbHVlcyA/Pz0gW107XG5cbiAgICB0aGlzLnZhbHVlcy5wdXNoKG5ldyBXaWtpQXJyYXlJdGVtKHVuZGVmaW5lZCwgdGhpcy52YWx1ZSkpO1xuICAgIHRoaXMudmFsdWUgPSB1bmRlZmluZWQ7XG4gIH1cblxuICBwdXNoKGl0ZW06IFdpa2lJdGVtKSB7XG4gICAgdGhpcy5jb252ZXJ0VG9BcnJheSgpO1xuXG4gICAgaWYgKGl0ZW0uYXJyYXkpIHtcbiAgICAgIHRoaXMudmFsdWVzPy5wdXNoKC4uLihpdGVtLnZhbHVlcyA/PyBbXSkpO1xuICAgIH0gZWxzZSB7XG4gICAgICB0aGlzLnZhbHVlcz8ucHVzaChuZXcgV2lraUFycmF5SXRlbSh1bmRlZmluZWQsIGl0ZW0udmFsdWUpKTtcbiAgICB9XG4gIH1cbn1cbiIsICJpbXBvcnQgeyBwcmVmaXgsIHN1ZmZpeCB9IGZyb20gJy4vc2hhcmVkLmpzJztcbmltcG9ydCB0eXBlIHsgV2lraSwgV2lraUFycmF5SXRlbSwgV2lraU1hcCB9IGZyb20gJy4vdHlwZXMuanMnO1xuXG5jb25zdCBzdHJpbmdpZnlBcnJheSA9IChhcnI6IFdpa2lBcnJheUl0ZW1bXSB8IHVuZGVmaW5lZCkgPT4ge1xuICBpZiAoIWFycikge1xuICAgIHJldHVybiAnJztcbiAgfVxuICByZXR1cm4gYXJyLnJlZHVjZSgocHJlLCBpdGVtKSA9PiBgJHtwcmV9XFxuWyR7aXRlbS5rID8gYCR7aXRlbS5rfXxgIDogJyd9JHtpdGVtLnYgPz8gJyd9XWAsICcnKTtcbn07XG5cbmV4cG9ydCBmdW5jdGlvbiBzdHJpbmdpZnkod2lraTogV2lraSk6IHN0cmluZyB7XG4gIGNvbnN0IGJvZHkgPSB3aWtpLmRhdGEucmVkdWNlKChwcmUsIGl0ZW0pID0+IHtcbiAgICBpZiAoaXRlbS5hcnJheSkge1xuICAgICAgcmV0dXJuIGAke3ByZX1cXG58JHtpdGVtLmtleX0gPSB7JHtzdHJpbmdpZnlBcnJheShpdGVtLnZhbHVlcyl9XFxufWA7XG4gICAgfVxuICAgIHJldHVybiBgJHtwcmV9XFxufCR7aXRlbS5rZXl9ID0gJHtpdGVtLnZhbHVlID8/ICcnfWA7XG4gIH0sICcnKTtcblxuICBjb25zdCB0eXBlID0gd2lraS50eXBlID8gJyAnICsgd2lraS50eXBlIDogJyc7XG5cbiAgcmV0dXJuIGAke3ByZWZpeH0ke3R5cGV9JHtib2R5fVxcbiR7c3VmZml4fWA7XG59XG5cbmV4cG9ydCBmdW5jdGlvbiBzdHJpbmdpZnlNYXAod2lraTogV2lraU1hcCk6IHN0cmluZyB7XG4gIGNvbnN0IGJvZHkgPSBbLi4ud2lraS5kYXRhXVxuICAgIC5tYXAoKFtrZXksIHZhbHVlXSkgPT4ge1xuICAgICAgaWYgKHR5cGVvZiB2YWx1ZSA9PT0gJ3N0cmluZycpIHtcbiAgICAgICAgcmV0dXJuIGB8JHtrZXl9ID0gJHt2YWx1ZSA/PyAnJ31gO1xuICAgICAgfVxuXG4gICAgICByZXR1cm4gYHwke2tleX0gPSB7JHtzdHJpbmdpZnlBcnJheSh2YWx1ZSl9XFxufWA7XG4gICAgfSlcbiAgICAuam9pbignXFxuJyk7XG5cbiAgY29uc3QgdHlwZSA9IHdpa2kudHlwZSA/ICcgJyArIHdpa2kudHlwZSA6ICcnO1xuXG4gIHJldHVybiBgJHtwcmVmaXh9JHt0eXBlfVxcbiR7Ym9keX1cXG4ke3N1ZmZpeH1gO1xufVxuIl0sCiAgIm1hcHBpbmdzIjogIjs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7O0FBQUE7QUFBQTtBQUFBO0FBQUE7QUFBQTtBQUFBO0FBQUE7QUFBQTtBQUFBO0FBQUE7QUFBQTtBQUFBO0FBQUE7QUFBQTtBQUFBO0FBQUE7QUFBQTtBQUFBOzs7QUNBTyxNQUFNLG9CQUFvQjtBQUMxQixNQUFNLG9CQUFvQjtBQUMxQixNQUFNLG9CQUFvQjtBQUMxQixNQUFNLHdCQUF3QjtBQUM5QixNQUFNLHlCQUF5QjtBQUMvQixNQUFNLDBCQUEwQjtBQUVoQyxNQUFNLGtC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
