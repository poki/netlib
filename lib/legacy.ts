import * as netlib from './index';

let target: any =
  "undefined" != typeof globalThis
    ? globalThis
    : "undefined" != typeof self
      ? self
      : "undefined" != typeof window
        ? window
        : "undefined" != typeof global
          ? global
          : {};

target["netlib"] = netlib;
