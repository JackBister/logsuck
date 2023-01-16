/**
 * Configuration for logsuck
 */
export interface LogsuckConfig {
  /**
   * A string containing the URL to this schema. Most likely "https://github.com/jackbister/logsuck/logsuck-config.schema.json", or "./logsuck-config.schema.json" if developing logsuck.
   */
  $schema?: string;
  /**
   * If enabled, the JSON configuration file will be used instead of the configuration saved in the database. This means that you cannot alter configuration at runtime and must instead update the JSON file and restart logsuck. Has no effect in forwarder mode. Default false.
   */
  forceStaticConfig?: boolean;
  /**
   * A fileType combines configuration related to a type of file. For example you may have certain config that is only applicable to access logs, in which case you might name a fileType "access_log" and put access log specific config there. The special fileType "DEFAULT" is applied to all files. In a forwarder/recipient setup this only needs to be configured on the recipient host.
   */
  fileTypes?: {
    /**
     * The name of this file type. Must be unique.
     */
    name?: string;
    /**
     * The layout of the _time field which will be extracted from this file. If no _time field is extracted or it doesn't match this layout, the time when the event was read will be used as the timestamp for that event. Default '2006/01/02 15:04:05'.
     */
    timeLayout?: string;
    /**
     * The duration between checking the file for updates. A low value will make the events searchable sooner at the cost of using more CPU and doing more disk reads. Default '1s'.
     */
    readInterval?: string;
    parser?: {
      /**
       * The name of the parser to use for this file. Default 'Regex'. Regex uses regular expressions to delimit events and extract values from them.
       */
      type?: "Regex";
      /**
       * Configuration specific to the Regex parser
       */
      regexConfig?: {
        /**
         * A regex specifying the delimiter between events. For example, if the file contains one event per row this should be '\n'. Default '\n'.
         */
        eventDelimiter?: string;
        /**
         * Regular expressions which will be used to extract field values from events.
         * Can be given in two variants:
         * 1. An expression containing any number of named capture groups. The names of the capture groups will be used as the field names and the captured strings will be used as the values.
         * 2. An expression with two unnamed capture groups. The first capture group will be used as the field name and the second group as the value.
         * If a field with the name '_time' is extracted and matches the given timelayout, it will be used as the timestamp of the event. Otherwise the time the event was read will be used.
         * Multiple extractors can be specified by using the fieldextractor flag multiple times. Defaults "(\w+)=(\w+)" and "(?P<_time>\d\d\d\d/\d\d/\d\d \d\d:\d\d:\d\d\.\d\d\d\d\d\d)")
         */
        fieldExtractors?: string[];
        [k: string]: unknown;
      };
      [k: string]: unknown;
    };
  }[];
  files?: {
    /**
     * The name of the file. This can also be a glob pattern such as "log-*.txt".
     */
    fileName?: string;
    fileTypes?: string[];
  }[];
  /**
   * A hostType contains configuration related to a type of host. For example your web server hosts may have different configuration than your database server hosts. The special hostType "DEFAULT" is applied to all hosts. In a forwarder/recipient setup this only needs to be configured on the recipient host.
   */
  hostTypes?: {
    /**
     * The name of the host type. Must be unique.
     */
    name?: string;
    /**
     * The files which should be indexed.
     */
    files?: {
      /**
       * The name of the file. This can also be a glob pattern such as "log-*.txt".
       */
      fileName: string;
      [k: string]: unknown;
    }[];
  }[];
  /**
   * Configuration related to the current host machine.
   */
  host?: {
    /**
     * The name of the host running this instance of logsuck. If empty or unset, logsuck will attempt to retrieve the hostname from the operating system.
     */
    name?: string;
    /**
     * The type of this host. Must be a key of the "hostTypes" object. This will define what files will be read by this instance of logsuck.
     */
    type?: string;
  };
  /**
   * Configuration for running in forwarder mode, where events will be pushed to a recipient instance of logsuck instead of being saved locally.
   */
  forwarder?: {
    /**
     * Whether forwarding mode should be enabled or not. Default false.
     */
    enabled?: boolean;
    /**
     * If the forwarder is unable to reach the recipient, events will begin to queue up. maxBufferedEvents is the maximum size of that queue. If maxBufferedEvents is exceeded before the forwarder can reach the recipient again, events will be lost.
     */
    maxBufferedEvents?: number;
    /**
     * The URL where the recipient instance is running. Default 'localhost:8081'.
     */
    recipientAddress?: string;
    /**
     * How often the forwarder should poll for configuration updates from the recipient. Must be a string like '1m', '15s', etc. Default '1m'.
     */
    configPollInterval?: string;
  };
  /**
   * Configuration for running in recipient mode, where events will be rececived from other logsuck instances in forwarder mode instead of reading directly from the log files.
   */
  recipient?: {
    /**
     * Whether recipient mode should be enabled or not. Default false.
     */
    enabled?: boolean;
    /**
     * The addreess where the API endpoints that the forwarders will communicate with should be exposed. Default ':8081'.
     */
    address?: string;
  };
  /**
   * Configuration for the SQLite database where logsuck will store its data. In a forwarder/recipient setup this only needs to be configured on the recipient host.
   */
  sqlite?: {
    /**
     * The file name which will be used for the SQLite database. Default 'logsuck.db'.
     */
    fileName?: string;
    /**
     * Whether Logsuck should use 'true batch' mode or not. True batch is significantly faster at saving events on average, but is slower at handling duplicates and relies on SQLite behavior which may not be guaranteed. Default true.
     */
    trueBatch?: boolean;
  };
  /**
   * Configuration for tasks, which run periodically and perform maintenance tasks like removing old events. If there is no configuration for a task it will never run.
   */
  tasks?: {
    /**
     * An array of configurations for each task.
     */
    tasks?: {
      /**
       * The name of the task.
       */
      name: string;
      /**
       * Whether the task should run or not.
       */
      enabled: boolean;
      /**
       * How often the task should run.
       */
      interval: string;
      /**
       * A key-value map from string to string of task-specific configuration. Check the documentation for a specific task to see which properties are available.
       */
      config?: {
        key?: string;
        value?: string;
        [k: string]: unknown;
      }[];
      [k: string]: unknown;
    }[];
  };
  /**
   * Configuration for the web GUI used to access logsuck. In a forwarder/recipient setup this only needs to be configured on the recipient host.
   */
  web?: {
    /**
     * Whether the web server should run. Defaults to true unless the configuration specifies that this logsuck instance should run in forwarder mode.
     */
    enabled?: boolean;
    /**
     * The address where the web server will be exposed. Default ':8080'.
     */
    address?: string;
    /**
     * If true, all static files will be served using the files that are bundled into the executable. If false, the normal filesystem will be used (which means the directory './internal/web/static/dist' must exist in the working directory). This is mostly useful when developing. Default true.
     */
    usePackagedFiles?: boolean;
    /**
     * Enables debug mode in the web server, which may enable features such as extra logging. Default false.
     */
    debugMode?: boolean;
  };
}
