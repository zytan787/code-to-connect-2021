import * as React from "react";
import Head from "next/head";
import styles from "../styles/Home.module.css";
import {
  Upload,
  Button,
  message,
  Row,
  Col,
  Spin,
  Tree,
  Result,
  Statistic,
} from "antd";
import {
  ArrowUpOutlined,
  ArrowDownOutlined,
  SendOutlined,
  DownloadOutlined,
  InboxOutlined,
} from "@ant-design/icons";
import { RcFile, UploadChangeParam } from "antd/es/upload";
import { useEffect, useMemo, useState } from "react";
import { InputFile } from "../api/compression/types";
import { useRequest } from "ahooks";
import { compressTradesRequest } from "../api/compression/api";
import { Base64 } from "js-base64";
import JSZip from "jszip";
import FileSaver from "file-saver";
import { UploadFile } from "antd/es/upload/interface";
import { Bar } from "react-chartjs-2";

const { Dragger } = Upload;

type OutputFile = {
  key: string;
  fileContent: string;
};

export default function Home() {
  const [inputFiles, setInputFiles] = useState<InputFile[]>([]);
  const [outputFiles, setOutputFiles] = useState<OutputFile[] | undefined>([]);
  const [checkedKeys, setCheckedKeys] = useState<string[] | undefined>(
    undefined
  );
  const [sentCompressRequest, setSentCompressRequest] = useState(false);
  const [preparingDownloads, setPreparingDownloads] = useState(false);

  const { data, error, loading, run } = useRequest(compressTradesRequest);

  const sendCompressTradesRequest = () => {
    run({ input_files: inputFiles });
    setSentCompressRequest(true);
  };

  const checkFileType = (file: RcFile): string => {
    if (file.type !== "text/csv") {
      message.error(`${file.name} is not a csv file`);
      return Upload.LIST_IGNORE;
    }
    return "";
  };

  const onUploadFileStatusChange = (info: UploadChangeParam) => {
    if (info.file.status === "done") {
      info.file.originFileObj
        ?.text()
        .then((value) => {
          const tmpInputFiles = inputFiles;
          tmpInputFiles.push({
            file_name: info.file.name,
            file_content: Base64.btoa(value),
          });
          setInputFiles(tmpInputFiles);
          message.success(`${info.file.name} file uploaded successfully`);
        })
        .catch((reason) =>
          message.error(`fail to get content from ${info.file.name} as string`)
        );
    } else if (info.file.status === "error") {
      message.error(`${info.file.name} file upload failed.`);
    }
  };

  const onRemove = (file: UploadFile) => {
    const tmpInputFiles = inputFiles;
    for (let i = 0; i < tmpInputFiles.length; i++) {
      if (file.name === tmpInputFiles[i].file_name) {
        tmpInputFiles.splice(i, 1);
        break;
      }
    }
    setInputFiles(tmpInputFiles);
  };

  useEffect(() => {
    if (data !== undefined && data.error === undefined) {
      const outputFiles: OutputFile[] = [];

      // outputFiles.push({
      //   key: "exclusion.csv",
      //   fileContent: new Blob([Base64.atob(data.exclusion)], { type: "text/csv" }),
      // });

      outputFiles.push({
        key: "exclusion.csv",
        fileContent: Base64.atob(data.exclusion),
      });

      outputFiles.push({
        key: "compression_report.csv",
        fileContent: Base64.atob(data.compression_report),
      });

      outputFiles.push({
        key: "compression_report_bookLevel.csv",
        fileContent: Base64.atob(data.compression_report_book_level),
      });

      for (let i = 0; i < data.proposals.length; i++) {
        outputFiles.push({
          key: `party_${data.proposals[i].party}_proposal.csv`,
          fileContent: Base64.atob(data.proposals[i].proposal),
        });
      }

      outputFiles.push({
        key: "data_check.csv",
        fileContent: Base64.atob(data.data_check),
      });

      setOutputFiles(outputFiles);
    } else {
      setOutputFiles(undefined);
    }
  }, [data]);

  const treeData = useMemo(() => {
    if (data !== undefined && data.error === undefined) {
      const proposals = [];
      const initialCheckedKeys = [];
      for (let i = 0; i < data.proposals.length; i++) {
        proposals.push({
          title: `party_${data.proposals[i].party}_proposal.csv`,
          key: `party_${data.proposals[i].party}_proposal.csv`,
        });
        initialCheckedKeys.push(
          `party_${data.proposals[i].party}_proposal.csv`
        );
      }

      const compare = (a: any, b: any) => {
        if (a.title < b.title) {
          return -1;
        }
        if (a.title > b.title) {
          return 1;
        }
        return 0;
      };

      proposals.sort(compare);

      initialCheckedKeys.push("exclusion.csv");
      initialCheckedKeys.push("compression_report.csv");
      initialCheckedKeys.push("compression_report_bookLevel.csv");
      initialCheckedKeys.push("data_check.csv");

      setCheckedKeys(initialCheckedKeys);

      return [
        {
          title: "all",
          key: "all",
          children: [
            { title: "exclusion.csv", key: "exclusion.csv" },
            { title: "compression_report.csv", key: "compression_report.csv" },
            {
              title: "compression_report_bookLevel.csv",
              key: "compression_report_bookLevel.csv",
            },
            { title: "proposals", key: "proposals", children: proposals },
            { title: "data_check.csv", key: "data_check.csv" },
          ],
        },
      ];
    } else {
      return undefined;
    }
  }, [data]);

  const notionalPercentage = useMemo(() => {
    if (data !== undefined && data.error === undefined) {
      let originalTotalNotional = 0;
      let newTotalNotional = 0;

      for (let i = 0; i < data.statistics.length; i++) {
        originalTotalNotional += data.statistics[i].original_notional;
        newTotalNotional += data.statistics[i].new_notional;
      }

      const percentage =
        ((originalTotalNotional - newTotalNotional) / originalTotalNotional) *
        100;

      return percentage;
    } else {
      return undefined;
    }
  }, [data]);

  const tradeCountPercentage = useMemo(() => {
    if (data !== undefined && data.error === undefined) {
      let originalTradeCount = 0;
      let newTradeCount = 0;

      for (let i = 0; i < data.statistics.length; i++) {
        originalTradeCount += data.statistics[i].original_no_of_trades;
        newTradeCount += data.statistics[i].new_no_of_trades;
      }

      const percentage =
        ((originalTradeCount - newTradeCount) / originalTradeCount) * 100;

      return percentage;
    } else {
      return undefined;
    }
  }, [data]);

  const notionalStatisticsData = useMemo(() => {
    if (data !== undefined && data.error === undefined) {
      const labels = [];
      const originalNotionals = [];
      const notionals = [];
      for (let i = 0; i < data.statistics.length; i++) {
        labels.push(data.statistics[i].party);
        originalNotionals.push(data.statistics[i].original_notional);
        notionals.push(data.statistics[i].new_notional);
      }

      const datasets = [];
      datasets.push({
        label: "Original Notional",
        backgroundColor: "#d3f261",
        data: originalNotionals,
      });
      datasets.push({
        label: "New Notional",
        backgroundColor: "#87e8de",
        data: notionals,
      });

      return {
        labels: labels,
        datasets: datasets,
      };
    } else {
      return undefined;
    }
  }, [data]);

  const notionalBarChartOptions = {
    indexAxis: "y",
    responsive: true,
    plugins: {
      legend: {
        position: "top",
      },
      title: {
        display: true,
        text: "Notional",
      },
    },
  };

  const tradeCountStatisticsData = useMemo(() => {
    if (data !== undefined && data.error === undefined) {
      const labels = [];
      const originalTradeCounts = [];
      const newTradeCounts = [];
      for (let i = 0; i < data.statistics.length; i++) {
        labels.push(data.statistics[i].party);
        originalTradeCounts.push(data.statistics[i].original_no_of_trades);
        newTradeCounts.push(data.statistics[i].new_no_of_trades);
      }

      const datasets = [];
      datasets.push({
        label: "Original Trade Count",
        backgroundColor: "#d3f261",
        data: originalTradeCounts,
      });
      datasets.push({
        label: "New Trade Count",
        backgroundColor: "#87e8de",
        data: newTradeCounts,
      });

      return {
        labels: labels,
        datasets: datasets,
      };
    } else {
      return undefined;
    }
  }, [data]);

  const tradeCountBarChartOptions = {
    indexAxis: "y",
    responsive: true,
    plugins: {
      legend: {
        position: "top",
      },
      title: {
        display: true,
        text: "Trade Count",
      },
    },
  };

  const onCheck = (
    checked: React.Key[] | { checked: React.Key[]; halfChecked: React.Key[] },
    info: any
  ) => {
    checked = checked as React.Key[];
    const checkedKeysStr = [];
    for (let i = 0; i < checked.length; i++) {
      checkedKeysStr.push(checked[i].toString());
    }
    setCheckedKeys(checkedKeysStr);
  };

  const downloadOutputFiles = () => {
    setPreparingDownloads(true);
    const zip = new JSZip();
    if (outputFiles !== undefined) {
      for (let i = 0; i < outputFiles.length; i++) {
        if (checkedKeys?.includes(outputFiles[i].key)) {
          zip.file(outputFiles[i].key, outputFiles[i].fileContent);
        }
      }
      zip.generateAsync({ type: "blob" }).then(function (content) {
        FileSaver.saveAs(content, "output.zip");
        setPreparingDownloads(false);
      });
    }
  };

  return (
    <div className={styles.container}>
      <Head>
        <title>Trade Compressor</title>
        <link rel="icon" href="/favicon.ico" />
      </Head>

      <h2 className={styles.title}>Trade Compressor</h2>

      <Row
        gutter={16}
        justify={"center"}
        style={{
          marginTop: "5em",
          textAlign: "center",
        }}
      >
        <Col span={8} style={{ textAlign: "center" }}>
          <div className={styles.area}>
            <h2>Input</h2>
            <Row justify={"center"}>
              <Col span={20}>
                <Dragger
                  beforeUpload={checkFileType}
                  multiple={true}
                  onChange={onUploadFileStatusChange}
                  onRemove={onRemove}
                >
                  <p className="ant-upload-drag-icon">
                    <InboxOutlined />
                  </p>
                  <p className="ant-upload-text">
                    Click or drag file to this area to upload
                  </p>
                  <p className="ant-upload-hint">
                    Support for a single or bulk upload
                  </p>
                </Dragger>
              </Col>
            </Row>
          </div>
        </Col>
        <Col span={4}>
          <Button
            type="primary"
            icon={<SendOutlined />}
            onClick={sendCompressTradesRequest}
            disabled={loading}
          >
            Compress Trades
          </Button>
        </Col>
        <Col span={8} style={{ textAlign: "center" }}>
          <div className={styles.area}>
            <h2>Output</h2>
            {loading && <Spin tip="Compressing..." />}
            {sentCompressRequest && !loading && error !== undefined && (
              <Result
                status="error"
                subTitle={error.message}
                // subTitle={data?.error}
              />
            )}
            {sentCompressRequest &&
              !loading &&
              error === undefined &&
              treeData !== undefined && (
                <>
                  <p>Select files to download:</p>
                  <Tree
                    checkable={true}
                    defaultExpandAll={true}
                    onCheck={onCheck}
                    checkedKeys={checkedKeys}
                    treeData={treeData}
                    style={{ backgroundColor: "#e6fffb", marginBottom: "1em" }}
                  />
                  <Button
                    type="primary"
                    icon={<DownloadOutlined />}
                    onClick={downloadOutputFiles}
                    disabled={
                      checkedKeys === undefined || checkedKeys.length === 0
                    }
                    loading={preparingDownloads}
                  >
                    Download
                  </Button>
                </>
              )}
          </div>
        </Col>
      </Row>

      <br />
      {!loading && notionalStatisticsData !== undefined && (
        <>
          <h2 style={{ textAlign: "center" }}>Statistics</h2>
          <Row justify={"center"}>
            {notionalPercentage !== undefined && (
              <Col span={8} style={{ textAlign: "right" }}>
                <Statistic
                  title="Notional"
                  value={
                    notionalPercentage > 0
                      ? notionalPercentage
                      : -notionalPercentage
                  }
                  precision={2}
                  valueStyle={{
                    color: notionalPercentage > 0 ? "#3f8600" : "#cf1322",
                  }}
                  prefix={
                    notionalPercentage > 0 ? (
                      <ArrowDownOutlined />
                    ) : (
                      <ArrowUpOutlined />
                    )
                  }
                  suffix="%"
                />
              </Col>
            )}
            {tradeCountPercentage !== undefined && (
              <Col offset={2} span={8}>
                <Statistic
                  title="Trade Count"
                  value={
                    tradeCountPercentage > 0
                      ? tradeCountPercentage
                      : -tradeCountPercentage
                  }
                  precision={2}
                  valueStyle={{
                    color: tradeCountPercentage > 0 ? "#3f8600" : "#cf1322",
                  }}
                  prefix={
                    tradeCountPercentage > 0 ? (
                      <ArrowDownOutlined />
                    ) : (
                      <ArrowUpOutlined />
                    )
                  }
                  suffix="%"
                />
              </Col>
            )}
          </Row>
          <br />
          <Row justify={"center"}>
            <Col span={16}>
              <Bar
                data={notionalStatisticsData}
                options={notionalBarChartOptions}
              />
            </Col>
          </Row>
          <br />
          <br />
          <Row justify={"center"}>
            <Col span={16}>
              <Bar
                data={tradeCountStatisticsData}
                options={tradeCountBarChartOptions}
              />
            </Col>
          </Row>
        </>
      )}
    </div>
  );
}
