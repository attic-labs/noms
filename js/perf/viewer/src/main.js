// @flow

// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

/* global Chart */

import {
  Dataset,
  DatasetSpec,
  invariant,
  List,
  Map as NomsMap,
  Struct,
} from '@attic/noms';
import type {Value} from '@attic/noms';

window.onload = load;
window.onpopstate = load;
window.onresize = render;

// The maximum number of git revisions to show in the perf history.
// The larger this number, the more screen real estate needed to render the graph - and the slower
// it will take to render, since the entire parent commit chain must be walked to form the graph.
// TODO: Implement paging mechanism.
const MAX_PERF_HISTORY = 15;

let chartDatasets: Map<string /* test name */, (number | null)[] /* elapsed, in seconds */>;
let chartLabels: string[];

async function load() {
  const params = getParams();
  if (!params.ds) {
    window.alert('Must provide a ?ds= param');
    return;
  }

  if (params.refresh) {
    window.setTimeout(() => window.location.reload(), params.refresh);
  }

  const dsSpec = DatasetSpec.parse(params.ds);
  const [perfData, gitRevs] = await getPerfHistory(dsSpec.dataset());

  chartDatasets = new Map();
  chartLabels = gitRevs.map(rev => rev.slice(0, 6));

  // Gather all test names up ahead of time, since some commits may be missing results.
  const testNames = uniq(flatten(await Promise.all(perfData.map(pd => {
    invariant(pd.reps instanceof List);
    // All reps will have the same set of test names.
    return pd.reps.get(0).then(firstRep => keys(firstRep));
  }))));

  const getElapsed = async (testName: string, pd: Struct) => {
    invariant(pd.reps instanceof List);
    const reps = await pd.reps.toJS();
    const elapsedOrNulls = await Promise.all(reps.map(rep => {
      invariant(rep instanceof NomsMap);
      // Note: despite how this code is structured, either all reps should have test data for this
      // value, or none should. Ideally we'd be able to bail at this point.
      return rep.get(testName).then(d => d !== null && d !== undefined ? d.elapsed / 1e9 : null);
    }));
    return elapsedOrNulls[0] !== null ? median(elapsedOrNulls) : null;
  };

  const getChartData = (testName: string) =>
    Promise.all(perfData.map(pd => getElapsed(testName, pd)));

  const testChartData = await Promise.all(testNames.map(getChartData));
  for (let i = 0; i < testNames.length; i++) {
    chartDatasets.set(testNames[i], testChartData[i]);
  }

  render();
}

// Returns the history of perf data with their git revisions, from oldest to newest.
async function getPerfHistory(ds: Dataset): Promise<[Struct[], string[]]> {
  const perfData = [], gitRevs = [];

  for (let head = await ds.head(), i = 0; head && i < MAX_PERF_HISTORY; i++) {
    const val = head.value;
    invariant(val instanceof Struct);
    perfData.push(val);
    gitRevs.push(val.nomsRevision);

    const parentRef = await head.parents.first(); // TODO: how to deal with multiple parents?
    head = parentRef ? await parentRef.targetValue(ds.database) : null;
  }

  return [perfData, gitRevs];
}

// Returns a map of URL param key to value.
function getParams(): {[key: string]: string} {
  // Note: this way anything after the # will end up in `params`, which is what we want.
  const params = {};
  const paramsIdx = location.href.indexOf('?');
  if (paramsIdx > -1) {
    decodeURIComponent(location.href.slice(paramsIdx + 1)).split('&').forEach(pair => {
      const [k, v] = pair.split('=');
      params[k] = v;
    });
  }
  return params;
}

async function render() {
  if (!chartDatasets) {
    return;
  }

  const datasets = [];
  for (const [testName, elapsed] of chartDatasets) {
    const [backgroundColor, borderColor] = await genLightAndDarkColors(testName);
    datasets.push({
      backgroundColor,
      borderColor,
      borderWidth: 1,
      data: elapsed,
      label: testName,
    });
  }

  // $FlowIssue: Chart is in modules, not imported, so Flow doesn't know about it.
  new Chart(document.getElementById('chart'), {
    type: 'line',
    data: {
      labels: chartLabels,
      datasets,
    },
    options: {
      scales: {
        yAxes: [{
          scaleLabel: {
            display: true,
            labelString: 'elapsed (seconds)',
          },
          ticks: {
            beginAtZero: true,
          },
        }],
        xAxes: [{
          scaleLabel: {
            display: true,
            labelString: 'github.com/attic-labs/noms git revision',
          },
        }],
      },
    },
  });
}

// Returns the median of numbers in `nums`.
function median(nums: number[]): number {
  const sorted = nums.slice();
  sorted.sort();
  const lenDiv2 = Math.floor(nums.length / 2);
  let res = nums[lenDiv2];
  if (nums.length % 2 === 0) {
    res += nums[lenDiv2 - 1];
    res /= 2;
  }
  return res;
}

// Generates a light and dark version of some color randomly (but stable) derived from `str`.
async function genLightAndDarkColors(str: string): Promise<[string, string]> {
  const strBuf = new window.TextEncoder().encode(str);
  const hash = await window.crypto.subtle.digest('sha-256', strBuf);
  invariant(hash instanceof ArrayBuffer);
  const [r, g, b] = new Uint8Array(hash);
  const [dr, dg, db] = [r, g, b].map(c => Math.ceil(c / 2));
  return [`rgba(${r}, ${g}, ${b}, 0.3)`, `rgb(${dr}, ${dg}, ${db})`];
}

// Returns the keys of `map`.
function keys<K: Value, V: Value>(map: NomsMap<K, V>): Promise<K[]> {
  return new Promise(res => {
    const keys = [];
    map.forEach((_, key) => {
      keys.push(key);
      return;
    }).then(() => res(keys));
  });
}

// Returns a single flat array from every array in `arrs` concatenated.
function flatten<T>(arrs: (T[])[]): T[] {
  return arrs.reduce((arr, res) => res.concat(arr), []);
}

// Returns `arr` with all duplicate values returned, without preserving order.
function uniq<T>(arr: T[]): T[] {
  // $FlowIssue: doesn't like my sneaky uniq.
  return [...new Set(arr)];
}
