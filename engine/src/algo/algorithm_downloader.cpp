#include "algo/algorithm_downloader.h"
#include "algo/algo_manager.h"
#include "algo/algo_utils.h"
#include <algorithm>
#include <cstdlib>
#include <curl/curl.h>
#include <filesystem>
#include <fstream>
#include <iomanip>
#include <iostream>
#include <mutex>
#include <openssl/md5.h>
#include <spawn.h>
#include <sstream>
#include <sys/wait.h>
#include <thread>
#include <unistd.h>

extern char **environ;

namespace aivision {
namespace algo {

void AlgorithmDownloader::StartDeploy(
    AlgoManager *algo_mgr, const std::string &download_url,
    const std::string &expected_md5, const std::string &extract_path,
    const std::string &algo_package_id, const std::string &algo_name,
    const std::string &version, const std::string &algo_dir) {
  std::thread t(DoDeploy, algo_mgr, download_url, expected_md5, extract_path,
                algo_package_id, algo_name, version, algo_dir);
  t.detach();
}

void AlgorithmDownloader::DoDeploy(
    AlgoManager *algo_mgr, std::string download_url, std::string expected_md5,
    std::string /*extract_path*/, std::string algo_package_id,
    std::string algo_name, std::string version, std::string algo_dir) {
  static std::mutex s_deploy_mutex;
  std::lock_guard<std::mutex> lock(s_deploy_mutex);

  // Use algo_dir from engine config instead of the path from heartbeat response
  std::string extract_path = algo_dir + "/" + algo_name + "_" + version;

  std::cout << "[AlgorithmDownloader] Starting deployment of " << algo_name
            << " (package " << algo_package_id << ")" << std::endl;

  algo_mgr->UpdateDeploymentStatus(algo_package_id, algo_name, version,
                                   extract_path, "downloading");

  // Secure temporary path for downloaded tar.gz
  char temp_template[] = "/tmp/aivision_algo_XXXXXX";
  int temp_fd = mkstemp(temp_template);
  if (temp_fd == -1) {
    std::cerr << "[AlgorithmDownloader] Failed to create temp file"
              << std::endl;
    algo_mgr->UpdateDeploymentStatus(algo_package_id, algo_name, version,
                                     extract_path, "failed",
                                     "创建临时文件失败");
    return;
  }
  close(temp_fd);
  std::string temp_tar = temp_template;

  // 1. Download
  if (!DownloadFile(download_url, temp_tar)) {
    std::cerr << "[AlgorithmDownloader] Failed to download algorithm package"
              << std::endl;
    algo_mgr->UpdateDeploymentStatus(algo_package_id, algo_name, version,
                                     extract_path, "failed", "下载算法包失败");
    return;
  }

  // 2. MD5 Verification
  if (!VerifyMD5(temp_tar, expected_md5)) {
    std::cerr << "[AlgorithmDownloader] MD5 verification failed" << std::endl;
    algo_mgr->UpdateDeploymentStatus(algo_package_id, algo_name, version,
                                     extract_path, "failed", "MD5 校验不匹配");
    std::filesystem::remove(temp_tar);
    return;
  }

  // 3. Extraction
  if (!ExtractTarGz(temp_tar, extract_path)) {
    std::cerr << "[AlgorithmDownloader] Extraction failed" << std::endl;
    algo_mgr->UpdateDeploymentStatus(algo_package_id, algo_name, version,
                                     extract_path, "failed", "解压算法包失败");
    std::filesystem::remove(temp_tar);
    return;
  }

  // Clean up temporary archive
  std::filesystem::remove(temp_tar);

  // 4. Resolve the required algorithm library from the fixed package layout
  std::string so_path = FindAlgorithmSo(extract_path);
  if (so_path.empty()) {
    std::cerr << "[AlgorithmDownloader] nikoniko_detector.so not found in "
              << extract_path << std::endl;
    algo_mgr->UpdateDeploymentStatus(algo_package_id, algo_name, version,
                                     extract_path, "failed",
                                     "未找到算法动态库文件");
    return;
  }

  std::cout << "[AlgorithmDownloader] Found algorithm library at: " << so_path
            << std::endl;

  // 5. Load the algorithm package
  auto instance = algo_mgr->Load(algo_name, version, so_path, "{}");
  if (!instance) {
    std::cerr
        << "[AlgorithmDownloader] Failed to load algorithm library into manager"
        << std::endl;
    algo_mgr->UpdateDeploymentStatus(algo_package_id, algo_name, version,
                                     extract_path, "failed",
                                     "加载算法动态库失败");
    return;
  }

  // Success!
  std::cout << "[AlgorithmDownloader] Deployment of " << algo_name
            << " successful!" << std::endl;
  algo_mgr->UpdateDeploymentStatus(algo_package_id, algo_name, version,
                                   extract_path, "installed");
}

bool AlgorithmDownloader::DownloadFile(const std::string &url,
                                       const std::string &local_path) {
  CURL *curl = curl_easy_init();
  if (!curl)
    return false;

  FILE *fp = fopen(local_path.c_str(), "wb");
  if (!fp) {
    curl_easy_cleanup(curl);
    return false;
  }

  curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
  curl_easy_setopt(curl, CURLOPT_WRITEDATA, fp);
  curl_easy_setopt(curl, CURLOPT_FAILONERROR, 1L);
  curl_easy_setopt(curl, CURLOPT_TIMEOUT, 120L);
  curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);

  CURLcode res = curl_easy_perform(curl);
  fclose(fp);

  if (res != CURLE_OK) {
    std::cerr << "[AlgorithmDownloader] CURL error: " << curl_easy_strerror(res)
              << " (code: " << res << ") URL: " << url << std::endl;
    curl_easy_cleanup(curl);
    std::filesystem::remove(local_path);
    return false;
  }

  // Check HTTP status code
  long http_code = 0;
  curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
  if (http_code >= 400) {
    std::cerr << "[AlgorithmDownloader] HTTP error code: " << http_code
              << " URL: " << url << std::endl;
    curl_easy_cleanup(curl);
    std::filesystem::remove(local_path);
    return false;
  }

  curl_easy_cleanup(curl);
  return true;
}

bool AlgorithmDownloader::VerifyMD5(const std::string &file_path,
                                    const std::string &expected_md5) {
  std::ifstream file(file_path, std::ios::binary);
  if (!file.is_open())
    return false;

  MD5_CTX md5Context;
  MD5_Init(&md5Context);

  char buffer[1024 * 16];
  while (file.good()) {
    file.read(buffer, sizeof(buffer));
    MD5_Update(&md5Context, buffer, file.gcount());
  }

  unsigned char result[MD5_DIGEST_LENGTH];
  MD5_Final(result, &md5Context);

  std::stringstream ss;
  for (int i = 0; i < MD5_DIGEST_LENGTH; i++) {
    ss << std::hex << std::setw(2) << std::setfill('0')
       << static_cast<int>(result[i]);
  }

  std::string calculated = ss.str();
  std::string expected = expected_md5;
  std::transform(expected.begin(), expected.end(), expected.begin(), ::tolower);

  return calculated == expected;
}

bool AlgorithmDownloader::ExtractTarGz(const std::string &tar_path,
                                       const std::string &dest_dir) {
  try {
    std::filesystem::create_directories(dest_dir);
  } catch (const std::exception &e) {
    std::cerr
        << "[AlgorithmDownloader] Failed to create destination directory: "
        << e.what() << std::endl;
    return false;
  }

  pid_t pid;
  const char *argv[] = {"tar", "-xzf",           tar_path.c_str(),
                        "-C",  dest_dir.c_str(), nullptr};
  int status;

  if (posix_spawnp(&pid, "tar", nullptr, nullptr, (char *const *)argv,
                   environ) != 0) {
    std::cerr << "[AlgorithmDownloader] posix_spawnp failed" << std::endl;
    return false;
  }

  if (waitpid(pid, &status, 0) == -1) {
    std::cerr << "[AlgorithmDownloader] waitpid failed" << std::endl;
    return false;
  }

  return WIFEXITED(status) && WEXITSTATUS(status) == 0;
}

} // namespace algo
} // namespace aivision
