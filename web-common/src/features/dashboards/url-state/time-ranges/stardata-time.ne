@preprocessor esmodule
@builtin "whitespace.ne"
@builtin "string.ne"

@{%
  import {
    StardataTime,

    StardataShorthandInterval,
    StardataPeriodToGrainInterval,
    StardataTimeStartEndInterval,
    StardataTimeOrdinalInterval,
    StardataIsoInterval,
    StardataLegacyIsoInterval,
    StardataLegacyDaxInterval,
    StardataAllTimeInterval,

    StardataPointInTime,
    StardataPointInTimeWithSnap,
    StardataLabelledPointInTime,
    StardataGrainPointInTime,
    StardataGrainPointInTimePart,
    StardataAbsoluteTime,
  } from "./StardataTime.ts"
%}

stardata_time => new_stardata_time {% id %}
           | old_stardata_time {% id %}

new_stardata_time => interval_with_grain                            {% id %}
               | interval_with_grain _ "tz" _ timezone_modifier {% ([rt, , , , tz]) => rt.withTimezone(tz) %}

interval_with_grain => interval_with_anchor_override _ "by"i _ grain {% ([rt, , , , grain]) => rt.withGrain(grain) %}
                     | interval_with_anchor_override                 {% id %}

interval_with_anchor_override => interval anchor_override:*      {% ([interval, anchorOverrides]) => new StardataTime(interval).withAnchorOverrides(anchorOverrides) %}
anchor_override               => _ "as"i _ "of"i _ point_in_time {% ([, , , , , pointInTime]) => pointInTime %}

interval => shorthand_interval         {% id %}
          | period_to_grain_interval   {% id %}
          | start_end_interval         {% id %}
          | ordinal_interval           {% id %}
          | iso_interval               {% id %}
          | "inf"i                     {% () => new StardataAllTimeInterval() %}

shorthand_interval => grain_duration {% ([parts]) => new StardataShorthandInterval(parts) %}

period_to_grain_interval => period_to_grain {% ([grain]) => new StardataPeriodToGrainInterval(grain) %}

ordinal_interval => ordinal (_ "of"i _ ordinal):* {% ([part, rest]) => new StardataTimeOrdinalInterval([part, ...rest.map(([, , , p]) => p)]) %}

start_end_interval => point_in_time _ "to"i _ point_in_time {% ([start, , , , end]) => new StardataTimeStartEndInterval(start, end) %}

iso_interval => abs_time _ "to"i _ abs_time {% ([start, , , , end]) => new StardataIsoInterval(start, end) %}
              | abs_time _ "/" _ abs_time   {% ([start, , , , end]) => new StardataIsoInterval(start, end) %}
              | abs_time _ "," _ abs_time   {% ([start, , , , end]) => new StardataIsoInterval(start, end) %}
              | abs_time                    {% ([start]) => new StardataIsoInterval(start, undefined) %}

point_in_time              => point_in_time_with_snap:* point_in_time_without_snap {% ([points, last]) => new StardataPointInTime([...points, last]) %}
                            | point_in_time_with_snap                              {% ([point]) => new StardataPointInTime([point]) %}
point_in_time_with_snap    => point_in_time_variants _ "/" _ grain _ "/" _ grain   {% ([point, , , , firstGrain, , , , secondGrain]) => new StardataPointInTimeWithSnap(point, [firstGrain, secondGrain]) %}
                            | point_in_time_variants _ "/" _ grain                 {% ([point, , , , grain]) => new StardataPointInTimeWithSnap(point, [grain]) %}
point_in_time_without_snap => point_in_time_variants                               {% ([point]) => new StardataPointInTimeWithSnap(point, []) %}

point_in_time_variants => grain_point_in_time   {% id %}
                        | labeled_point_in_time {% id %}
                        | abs_time              {% id %}

grain_point_in_time      => grain_point_in_time_part:+ {% ([parts]) => new StardataGrainPointInTime([...parts]) %}
grain_point_in_time_part => prefix _ grain_duration    {% ([prefix, _, grains]) => new StardataGrainPointInTimePart(prefix, grains) %}

labeled_point_in_time => "earliest"i  {% StardataLabelledPointInTime.postProcessor %}
                       | "latest"i    {% StardataLabelledPointInTime.postProcessor %}
                       | "now"i       {% StardataLabelledPointInTime.postProcessor %}
                       | "watermark"i {% StardataLabelledPointInTime.postProcessor %}
                       | "ref"i       {% StardataLabelledPointInTime.postProcessor %}

ordinal => grain num {% ([grain, num]) => ({num, grain}) %}

grain_duration      => grain_duration_part:+ {% ([parts]) => parts %}
grain_duration_part => num grain             {% ([num, grain]) => ({num, grain}) %}

period_to_grain => grain "TD" {% ([grain]) => grain %}

abs_time => [\d] [\d] [\d] [\d] [\-] [\d] [\d] [\-] [\d] [\d] "T" [\d] [\d] [:] [\d] [\d] [:] [\d] [\d] [.] [\d]:+ "Z" {% StardataAbsoluteTime.postProcessor %}
          | [\d] [\d] [\d] [\d] [\-] [\d] [\d] [\-] [\d] [\d] "T" [\d] [\d] [:] [\d] [\d] [:] [\d] [\d] "Z"            {% StardataAbsoluteTime.postProcessor %}
          | [\d] [\d] [\d] [\d] [\-] [\d] [\d] [\-] [\d] [\d] "T" [\d] [\d] [:] [\d] [\d]                              {% StardataAbsoluteTime.postProcessor %}
          | [\d] [\d] [\d] [\d] [\-] [\d] [\d] [\-] [\d] [\d] "T" [\d] [\d]                                            {% StardataAbsoluteTime.postProcessor %}
          | [\d] [\d] [\d] [\d] [\-] [\d] [\d] [\-] [\d] [\d]                                                          {% StardataAbsoluteTime.postProcessor %}
          | [\d] [\d] [\d] [\d] [\-] [\d] [\d]                                                                         {% StardataAbsoluteTime.postProcessor %}
          | [\d] [\d] [\d] [\d]                                                                                        {% StardataAbsoluteTime.postProcessor %}

timezone_modifier => [0-9a-zA-Z/+\-_]:+ {% ([args]) => args.join("") %}

old_stardata_time => iso_time {% ([legacyIso]) => new StardataTime(legacyIso) %}
               | dax_time {% ([legacyDax]) => new StardataTime(new StardataLegacyDaxInterval(legacyDax)) %}

iso_time => "P" iso_date_part:+ "T" iso_time_part:+ {% ([, dateGrains, , timeGrains]) => new StardataLegacyIsoInterval(dateGrains, timeGrains) %}
          | "P" iso_date_part:+                     {% ([, dateGrains]) => new StardataLegacyIsoInterval(dateGrains, []) %}
          | "PT" iso_time_part:+                    {% ([, timeGrains]) => new StardataLegacyIsoInterval([], timeGrains) %}

iso_date_part => num date_grains {% ([num, grain]) => ({num, grain}) %}
iso_time_part => num time_grains {% ([num, grain]) => ({num, grain}) %}

dax_time => "rill-" dax_notations    {% (args) => args.join("") %}
dax_notations => dax_to_date "TD"    {% (args) => args.join("") %}
               | "TD"                {% id %}
               | "P" date_grains "C" {% (args) => args.join("") %}
               | "PP"                {% id %}
               | "P" date_grains     {% (args) => args.join("") %}

prefix => [+\-] {% id %}

num => [0-9]:+ {% ([args]) => Number(args.join("")) %}

grain => [sSmhHdDwWqQMyY] {% id %}

date_grains => [DWQMY] {% id %}
time_grains => [SMH] {% id %}
dax_to_date => [WQMY] {% id %}
