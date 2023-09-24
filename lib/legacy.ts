import * as netlib from './index'

const target: any =
  typeof globalThis !== 'undefined'
    ? globalThis
    : typeof self !== 'undefined'
      ? self
      : typeof window !== 'undefined'
        ? window
        : typeof global !== 'undefined'
          ? global
          : {}

target.netlib = netlib
