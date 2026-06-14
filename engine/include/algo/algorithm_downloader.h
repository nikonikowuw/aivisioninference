#ifndef AIVISION_ALGO_ALGORITHM_DOWNLOADER_H
#define AIVISION_ALGO_ALGORITHM_DOWNLOADER_H

#include <string>

namespace aivision
{
    namespace algo
    {
        class AlgoManager;

        class AlgorithmDownloader
        {
        public:
            /// 启动异步线程下载并解压算法包
            static void StartDeploy(
                AlgoManager* algo_mgr,
                const std::string& download_url,
                const std::string& expected_md5,
                const std::string& extract_path,
                const std::string& algo_package_id,
                const std::string& algo_name,
                const std::string& version,
                const std::string& algo_dir = "/opt/aivision/algo"
            );

        private:
            /// 执行下载、校验、解压、加载
            static void DoDeploy(
                AlgoManager* algo_mgr,
                std::string download_url,
                std::string expected_md5,
                std::string extract_path,
                std::string algo_package_id,
                std::string algo_name,
                std::string version,
                std::string algo_dir
            );

            static bool DownloadFile(const std::string& url, const std::string& local_path);
            static bool VerifyMD5(const std::string& file_path, const std::string& expected_md5);
            static bool ExtractTarGz(const std::string& tar_path, const std::string& dest_dir);
        };
    }
}

#endif // AIVISION_ALGO_ALGORITHM_DOWNLOADER_H
