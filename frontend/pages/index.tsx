import * as React from "react";
import Head from "next/head";
import Image from "next/image";
import styles from "../styles/Home.module.css";
import {
  Upload,
  Button,
  message,
  Row,
  Col,
  Card,
  Spin,
  Tree,
  Result,
} from "antd";
import {
  UploadOutlined,
  SendOutlined,
  DownloadOutlined,
} from "@ant-design/icons";
import { RcFile, UploadChangeParam } from "antd/es/upload";
import { useEffect, useMemo, useState } from "react";
import { InputFile } from "./api/compression/types";
import { useRequest } from "ahooks";
import { compressTradesRequest } from "./api/compression/api";
import { Base64 } from "js-base64";
import JSZip from "jszip";
import FileSaver from "file-saver";

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

  const { data, error, loading, run } = useRequest(compressTradesRequest);

  const sendCompressTradesRequest = () => {
    if (inputFiles.length > 0) {
      run({ input_files: inputFiles });
    }
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
      for (let i = 0; i < data.proposals.length; i++) {
        proposals.push({
          title: `party_${data.proposals[i].party}_proposal.csv`,
          key: `party_${data.proposals[i].party}_proposal.csv`,
        });
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
    const zip = new JSZip();
    if (outputFiles !== undefined) {
      for (let i = 0; i < outputFiles.length; i++) {
        if (checkedKeys?.includes(outputFiles[i].key)) {
          zip.file(outputFiles[i].key, outputFiles[i].fileContent);
        }
      }
      zip.generateAsync({ type: "blob" }).then(function (content) {
        FileSaver.saveAs(content, "output.zip");
      });
    }
  };

  return (
    <div className={styles.container}>
      <Head>
        <title>Trade Compressor</title>
        <link rel="icon" href="/favicon.ico" />
      </Head>

      <h1 className={styles.title}>
        Welcome to <a href="https://nextjs.org">Next.js!</a>
      </h1>

      <p className={styles.description}>
        Get started by editing{" "}
        <code className={styles.code}>pages/index.js</code>
      </p>

      <Row gutter={16}>
        <Col span={10} style={{ textAlign: "center" }}>
          <h2>Input</h2>
          <Upload
            beforeUpload={checkFileType}
            multiple={true}
            onChange={onUploadFileStatusChange}
          >
            <Button icon={<UploadOutlined />}>Click to Upload</Button>
          </Upload>
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
        <Col span={10} style={{ textAlign: "center" }}>
          <h2>Output</h2>
          {loading && <Spin tip="Compressing..." />}
          {!loading && error !== undefined && (
            <Result
              status="error"
              title={`${error.name}: ${error.message}`}
              subTitle={data?.error}
            />
          )}
          {!loading && error === undefined && treeData !== undefined && (
            <>
              <Tree
                checkable={true}
                defaultExpandAll={true}
                onCheck={onCheck}
                checkedKeys={checkedKeys}
                treeData={treeData}
              />
              <Button
                type="primary"
                icon={<DownloadOutlined />}
                onClick={downloadOutputFiles}
                disabled={checkedKeys === undefined || checkedKeys.length === 0}
              >
                Download
              </Button>
            </>
          )}
        </Col>
      </Row>

      <footer className={styles.footer}>
        <a
          href="https://vercel.com?utm_source=create-next-app&utm_medium=default-template&utm_campaign=create-next-app"
          target="_blank"
          rel="noopener noreferrer"
        >
          Powered by{" "}
          <span className={styles.logo}>
            <Image src="/vercel.svg" alt="Vercel Logo" width={72} height={16} />
          </span>
        </a>
      </footer>
    </div>
  );
}
